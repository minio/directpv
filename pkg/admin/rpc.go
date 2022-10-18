// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package admin

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	pkgclient "github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	pkgdevice "github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	losetup "gopkg.in/freddierice/go-losetup.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func reflinkSupported(ctx context.Context) (bool, error) {
	var errMountFailed = errors.New("unable to mount")

	checkXFS := func(ctx context.Context, reflink bool) error {
		mountPoint, err := os.MkdirTemp("", "xfs.check.mnt.")
		if err != nil {
			return err
		}
		defer os.Remove(mountPoint)

		file, err := os.CreateTemp("", "xfs.check.file.")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		file.Close()

		if err = os.Truncate(file.Name(), xfs.MinSupportedDeviceSize); err != nil {
			return err
		}

		if _, _, _, _, err = xfs.MakeFS(ctx, file.Name(), uuid.New().String(), false, reflink); err != nil {
			return err
		}

		loopDevice, err := losetup.Attach(file.Name(), 0, false)
		if err != nil {
			return err
		}

		defer func() {
			if err := loopDevice.Detach(); err != nil {
				klog.Error(err)
			}
		}()

		if err = xfs.Mount(loopDevice.Path(), mountPoint); err != nil {
			return fmt.Errorf("%w; %v", errMountFailed, err)
		}

		return sys.Unmount(mountPoint, true, true, false)
	}

	reflinkSupport := true
	err := checkXFS(ctx, reflinkSupport)
	if err == nil {
		return reflinkSupport, nil
	}

	if !errors.Is(err, errMountFailed) {
		return false, err
	}

	reflinkSupport = false
	return reflinkSupport, checkXFS(ctx, reflinkSupport)
}

type nodeRPCServer struct {
	nodeID    directpvtypes.NodeID
	reflink   bool
	topology  map[string]string
	lockMap   map[string]*sync.Mutex
	lockMapMu sync.Mutex

	probeDevices func() ([]pkgdevice.Device, error)
	getDevices   func(majorMinor ...string) ([]pkgdevice.Device, error)
	getMounts    func() (map[string][]string, error)
	makeFS       func(device, fsuuid string, force, reflink bool) (string, string, uint64, uint64, error)
	mount        func(device, fsuuid string) error
	unmount      func(fsuuid string) error
	symlink      func(fsuuid string) error
	mkdir        func(fsuuid string) error
	writeFile    func(fsuuid, data string) error
}

func newNodeRPCServer(ctx context.Context, nodeID directpvtypes.NodeID, topology map[string]string) (*nodeRPCServer, error) {
	reflink, err := reflinkSupported(ctx)
	if err != nil {
		return nil, err
	}

	if reflink {
		klog.V(3).Infof("XFS reflink support is enabled")
	} else {
		klog.V(3).Infof("XFS reflink support is disabled")
	}

	return &nodeRPCServer{
		reflink:  reflink,
		nodeID:   nodeID,
		topology: topology,
		lockMap:  map[string]*sync.Mutex{},

		probeDevices: pkgdevice.Probe,
		getDevices:   pkgdevice.ProbeDevices,
		getMounts: func() (deviceMap map[string][]string, err error) {
			if _, deviceMap, err = sys.GetMounts(); err != nil {
				err = fmt.Errorf("unable get mount points; %w", err)
			}
			return
		},
		makeFS: func(device, fsuuid string, force, reflink bool) (string, string, uint64, uint64, error) {
			fsuuid, label, totalCapacity, freeCapacity, err := xfs.MakeFS(context.Background(), device, fsuuid, force, reflink)
			if err != nil {
				err = fmt.Errorf("unable to format device %v; %w", device, err)
			}
			return fsuuid, label, totalCapacity, freeCapacity, err
		},
		mount: func(device, fsuuid string) (err error) {
			if err = xfs.Mount(device, types.GetDriveMountDir(fsuuid)); err != nil {
				err = fmt.Errorf("unable to mount %v to %v; %w", device, types.GetDriveMountDir(fsuuid), err)
			}
			return
		},
		unmount: func(fsuuid string) (err error) {
			if err = sys.Unmount(types.GetDriveMountDir(fsuuid), true, true, false); err != nil {
				err = fmt.Errorf("unable to unmount %v; %w", types.GetDriveMountDir(fsuuid), err)
			}
			return
		},
		symlink: func(fsuuid string) (err error) {
			if err = os.Symlink(".", types.GetVolumeRootDir(fsuuid)); err != nil {
				err = fmt.Errorf("unable to create symlink %v; %w", types.GetVolumeRootDir(fsuuid), err)
			}
			return
		},
		mkdir: func(fsuuid string) (err error) {
			if err = os.Mkdir(types.GetDriveMetaDir(fsuuid), 0o750); err != nil {
				err = fmt.Errorf("unable to create meta directory %v; %w", types.GetDriveMetaDir(fsuuid), err)
			}
			return
		},
		writeFile: func(fsuuid, data string) (err error) {
			if err = os.WriteFile(types.GetDriveMetaFile(fsuuid), []byte(data), 0o640); err != nil {
				err = fmt.Errorf("unable to create meta file %v; %w", types.GetDriveMetaFile(fsuuid), err)
			}
			return
		},
	}, nil
}

func (server *nodeRPCServer) getLock(name string) *sync.Mutex {
	server.lockMapMu.Lock()
	defer server.lockMapMu.Unlock()

	mutex, found := server.lockMap[name]
	if !found {
		mutex = &sync.Mutex{}
		server.lockMap[name] = mutex
	}

	return mutex
}

// NodeListDevicesRequest is request arguments of /drives/list API.
type NodeListDevicesRequest struct {
	Devices       []string `json:"devices,omitempty"`
	FormatAllowed bool     `json:"formatAllowed,omitempty"`
	FormatDenied  bool     `json:"formatDenied,omitempty"`
}

// NodeListDevicesResponse is response of /drives/list API.
type NodeListDevicesResponse struct {
	Devices []Device `json:"devices,omitempty"`
}

func (server *nodeRPCServer) ListDevices(req *NodeListDevicesRequest) (resp *NodeListDevicesResponse, err error) {
	probedDevices, err := server.probeDevices()
	if err != nil {
		return nil, err
	}

	devices := []Device{}
	for i := range probedDevices {
		if len(req.Devices) != 0 && !utils.Contains(req.Devices, probedDevices[i].Name) {
			continue
		}

		d := newDevice(probedDevices[i])
		if (!req.FormatAllowed && !req.FormatDenied) || (req.FormatAllowed && !d.FormatDenied) || (req.FormatDenied && d.FormatDenied) {
			devices = append(devices, d)
		}
	}

	return &NodeListDevicesResponse{Devices: devices}, nil
}

// NodeFormatDevicesRequest is request arguments of /drives/format API.
type NodeFormatDevicesRequest struct {
	Devices []FormatDevice `json:"devices,omitempty"`
}

// FormatResult is drive format result
type FormatResult struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// NodeFormatDevicesResponse is response arguments of /drives/format API.
type NodeFormatDevicesResponse struct {
	Devices []FormatResult `json:"devices,omitempty"`
}

func (server *nodeRPCServer) format(mutex *sync.Mutex, device pkgdevice.Device, force, reflink bool) error {
	mutex.Lock()
	defer mutex.Unlock()

	devPath := utils.AddDevPrefix(device.Name)

	deviceMap, err := server.getMounts()
	if err != nil {
		return err
	}

	if mountPoints, found := deviceMap[devPath]; found {
		return fmt.Errorf("device %v mounted at %v", devPath, mountPoints)
	}

	fsuuid := uuid.New().String()

	_, _, totalCapacity, freeCapacity, err := server.makeFS(devPath, fsuuid, force, reflink)
	if err != nil {
		return err
	}

	if err = server.mount(devPath, fsuuid); err != nil {
		return err
	}
	defer func() {
		if err == nil {
			return
		}
		if uerr := server.unmount(fsuuid); uerr != nil {
			err = fmt.Errorf("%w; %v", err, uerr)
		}
	}()

	if err = server.symlink(fsuuid); err != nil {
		return err
	}

	if err = server.mkdir(fsuuid); err != nil {
		return err
	}

	data := fmt.Sprintf("APP_NAME=%v\nAPP_VERSION=%v\nFSUUID=%v\n", consts.AppName, consts.LatestAPIVersion, fsuuid)
	if err = server.writeFile(fsuuid, data); err != nil {
		return err
	}

	drive := types.NewDrive(
		directpvtypes.DriveID(fsuuid),
		types.DriveStatus{
			TotalCapacity: int64(totalCapacity),
			FreeCapacity:  int64(freeCapacity),
			FSUUID:        fsuuid,
			Status:        directpvtypes.DriveStatusReady,
			Make:          device.Make(),
			Topology:      server.topology,
		},
		server.nodeID,
		directpvtypes.DriveName(device.Name),
		directpvtypes.AccessTierDefault,
	)
	if _, err = pkgclient.DriveClient().Create(context.Background(), drive, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("unable to create Drive CRD; %w", err)
	}

	return nil
}

func (server *nodeRPCServer) FormatDevices(req *NodeFormatDevicesRequest) (resp *NodeFormatDevicesResponse, err error) {
	var majorMinorList []string
	for i := range req.Devices {
		majorMinorList = append(majorMinorList, req.Devices[i].MajorMinor)
	}

	devices, err := server.getDevices(majorMinorList...)
	if err != nil {
		return nil, err
	}

	probedDevices := map[string]pkgdevice.Device{}
	for _, device := range devices {
		probedDevices[device.MajorMinor] = device
	}

	results := make([]FormatResult, len(req.Devices))
	var wg sync.WaitGroup
	for i := range req.Devices {
		results[i].Name = req.Devices[i].Name
		device, found := probedDevices[req.Devices[i].MajorMinor]
		switch {
		case !found:
			results[i].Error = "device not found"
		case !device.Equal(req.Devices[i].Device):
			results[i].Error = "device state changed"
		default:
			mutex := server.getLock(device.Name)
			wg.Add(1)
			go func(i int, mutex *sync.Mutex, device pkgdevice.Device, force, reflink bool) {
				defer wg.Done()
				if err := server.format(mutex, device, force, reflink); err != nil {
					results[i].Error = err.Error()
				}
			}(i, mutex, device, req.Devices[i].Force, server.reflink)
		}
	}
	wg.Wait()

	return &NodeFormatDevicesResponse{Devices: results}, nil
}

type rpcServer struct {
	client *http.Client
}

func newRPCServer() *rpcServer {
	return &rpcServer{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 1 * time.Minute}).DialContext,
				MaxIdleConnsPerHost:   1024,
				IdleConnTimeout:       1 * time.Minute,
				ResponseHeaderTimeout: 1 * time.Minute,
				TLSHandshakeTimeout:   15 * time.Second,
				ExpectContinueTimeout: 3 * time.Second,
				TLSClientConfig: &tls.Config{
					// Can't use SSLv3 because of POODLE and BEAST
					// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
					// Can't use TLSv1.1 because of RC4 cipher usage
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: true, // FIXME: use trusted CA
				},
			},
		},
	}
}

func (server *rpcServer) getNodeClients() (map[string]*nodeClient, error) {
	endpoints, err := k8s.KubeClient().CoreV1().Endpoints(consts.Namespace).Get(context.Background(), consts.NodeAPIServerHLSVC, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if len(endpoints.Subsets) == 0 {
		return nil, fmt.Errorf("no subsets found in endpoints")
	}

	port := int32(0)
	for _, endpointPort := range endpoints.Subsets[0].Ports {
		if endpointPort.Name == consts.NodeAPIPortName {
			port = endpointPort.Port
			break
		}
	}
	if port == 0 {
		return nil, fmt.Errorf("port not found in endpoint subset")
	}

	nodeClients := make(map[string]*nodeClient)
	for _, address := range endpoints.Subsets[0].Addresses {
		nodeClients[*address.NodeName] = &nodeClient{
			url: &url.URL{
				Scheme: "https",
				Host:   fmt.Sprintf("%v:%v", address.IP, port),
				Path:   "/",
			},
			client: server.client,
		}
	}
	if len(nodeClients) == 0 {
		return nil, fmt.Errorf("no nodes found in endpoint subset")
	}

	return nodeClients, nil
}

// ListDevicesRequest is request arguments of /drives/list API.
type ListDevicesRequest struct {
	Nodes         []string `json:"nodes,omitempty"`
	Devices       []string `json:"devices,omitempty"`
	FormatAllowed bool     `json:"formatAllowed,omitempty"`
	FormatDenied  bool     `json:"formatDenied,omitempty"`
}

type ListDevicesResult struct {
	Devices []Device `json:"devices,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// ListDevicesResponse is response of /drives/list API.
type ListDevicesResponse struct {
	Nodes map[string]ListDevicesResult `json:"nodes,omitempty"`
	Error string                       `json:"error,omitempty"`
}

func (server *rpcServer) ListDevices(req *ListDevicesRequest) (resp *ListDevicesResponse, err error) {
	nodeClients, err := server.getNodeClients()
	if err != nil {
		return nil, err
	}

	resp = &ListDevicesResponse{
		Nodes: map[string]ListDevicesResult{},
	}

	nodes := []string{}
	if len(req.Nodes) != 0 {
		for _, node := range req.Nodes {
			if _, found := nodeClients[node]; found {
				nodes = append(nodes, node)
			}
		}
		if len(nodes) == 0 {
			return resp, nil
		}
	} else {
		for node := range nodeClients {
			nodes = append(nodes, node)
		}
	}

	mutex := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, nodeName := range nodes {
		wg.Add(1)
		go func(mutex *sync.Mutex, nodeName string, client *nodeClient) {
			defer wg.Done()
			results, err := client.ListDevices(req.Devices, req.FormatAllowed, req.FormatDenied)
			var e string
			if err != nil {
				e = err.Error()
			}

			mutex.Lock()
			resp.Nodes[nodeName] = ListDevicesResult{
				Devices: results,
				Error:   e,
			}
			mutex.Unlock()
		}(mutex, nodeName, nodeClients[nodeName])
	}
	wg.Wait()

	return resp, nil
}

// FormatDevicesRequest is request arguments of /drives/format API.
type FormatDevicesRequest struct {
	Nodes map[string][]FormatDevice `json:"nodes,omitempty"`
}

type FormatDevicesResult struct {
	Devices []FormatResult `json:"devices,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// FormatDevicesResponse is response arguments of /drives/format API.
type FormatDevicesResponse struct {
	Nodes map[string]FormatDevicesResult `json:"nodes,omitempty"`
	Error string                         `json:"error,omitempty"`
}

func (server *rpcServer) FormatDevices(req *FormatDevicesRequest) (resp *FormatDevicesResponse, err error) {
	nodeClients, err := server.getNodeClients()
	if err != nil {
		return nil, err
	}

	if len(req.Nodes) != 0 {
		var nodeNames []string
		for nodeName := range req.Nodes {
			if _, found := nodeClients[nodeName]; !found {
				nodeNames = append(nodeNames, nodeName)
			}
		}
		if len(nodeNames) != 0 {
			return &FormatDevicesResponse{
				Error: fmt.Sprintf("unknown nodes %v", nodeNames),
			}, nil
		}
	}

	resp = &FormatDevicesResponse{
		Nodes: map[string]FormatDevicesResult{},
	}
	mutex := &sync.Mutex{}

	var wg sync.WaitGroup
	for nodeName, devices := range req.Nodes {
		wg.Add(1)
		go func(mutex *sync.Mutex, nodeName string, client *nodeClient, devices []FormatDevice) {
			defer wg.Done()
			results, err := client.FormatDevices(devices)
			var e string
			if err != nil {
				e = err.Error()
			}

			mutex.Lock()
			resp.Nodes[nodeName] = FormatDevicesResult{
				Devices: results,
				Error:   e,
			}
			mutex.Unlock()
		}(mutex, nodeName, nodeClients[nodeName], devices)
	}
	wg.Wait()

	return resp, nil
}
