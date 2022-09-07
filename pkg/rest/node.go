// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

package rest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"sync"

	"github.com/google/uuid"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/matcher"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	nodeAPIPort  = "40443"
	nodeCertPath = "/tmp/certs/node-cert.pem"
	nodeKeyPath  = "/tmp/certs/node-key.pem"
)

// errors
var (
	errUDevDataMismatch  = errors.New("udev data isn't matching")
	errForceRequired     = errors.New("force flag is required for formatting")
	errDriveAlreadyAdded = errors.New("drive is already formatted and added")
	errDuplicateDevice   = errors.New("found duplicate devices for drive")
)

// suggestions
var (
	formatRetrySuggestion          = "retry the format request"
	formatRetryWithForceSuggestion = "retry the format request with force"
)

// reasons
var (
	udevDataNotAccessibleReason = "couldn't read the udev data"
	udevDataMismatchReason      = "probed udevdata isn't matching with the udev data in the request"
)

// ServeNodeAPIServer starts the DirectPV Node API server
func ServeNodeAPIServer(ctx context.Context, nodeID string) error {
	certs, err := tls.LoadX509KeyPair(nodeCertPath, nodeKeyPath)
	if err != nil {
		klog.Errorf("Filed to load key pair: %v", err)
		return err
	}

	// Create a secure http server
	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certs},
			InsecureSkipVerify: true,
		},
	}

	nodeHandler, err := newNodeAPIHandler(ctx, nodeID)
	if err != nil {
		return err
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc(drivesList, nodeHandler.listLocalDrivesHandler)
	mux.HandleFunc(drivesFormat, nodeHandler.formatLocalDrivesHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", nodeAPIPort))
	if lErr != nil {
		return lErr
	}

	go func() {
		klog.V(3).Infof("Starting DirectPV Node API server in port: %s", nodeAPIPort)
		if err := server.ServeTLS(listener, "", ""); err != nil {
			klog.Errorf("Failed to listen and serve DirectPV Node API server: %v", err)
		}
	}()

	return nil
}

// listLocalDrivesHandler fetches the devices present in the node and sends back
func (n *nodeAPIHandler) listLocalDrivesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Unmarshal API request
	var req GetDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	deviceList, err := n.listLocalDrives(context.Background(), req)
	if err != nil {
		klog.Errorf("couldn't list local drives: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Marshal API response
	jsonBytes, err := json.Marshal(GetDevicesResponse{
		DeviceInfo: map[NodeName][]Device{
			NodeName(n.nodeID): deviceList,
		},
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
	return
}

func (n *nodeAPIHandler) listLocalDrives(ctx context.Context, req GetDevicesRequest) ([]Device, error) {
	var drives, statuses []string
	var err error
	if len(req.Drives) > 0 {
		drives, err = ellipsis.Expand(string(req.Drives))
		if err != nil {
			return nil, fmt.Errorf("couldn't expand the node selector %v: %v", req.Nodes, err)
		}
	}
	for _, status := range req.Statuses {
		statuses = append(statuses, string(status))
	}
	// Fetch the
	devices, err := node.ProbeDevices()
	if err != nil {
		return nil, fmt.Errorf("couldn't probe the devices: %v", err)
	}
	localDirectPVDrives, err := n.listLocalDirectPVDrives(context.Background())
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch the local DirectPV drives: %v", err)
	}

	var deviceList []Device
	for _, directPVDrive := range localDirectPVDrives {
		matchedDevices, unmatchedDevices := getMatchedDevicesForDirectPVDrive(&directPVDrive, devices)
		switch len(matchedDevices) {
		case 0:
			// Device which was online before is lost/detached/corrupted now
			if len(statuses) > 0 && !matcher.StringIn(statuses, string(DeviceStatusOffline)) {
				break
			}
			deviceName := path.Base(directPVDrive.Status.Path)
			if len(drives) > 0 && !matcher.StringIn(drives, deviceName) {
				break
			}
			deviceList = append(deviceList, Device{
				Name:       deviceName,
				Major:      int(directPVDrive.Status.MajorNumber),
				Minor:      int(directPVDrive.Status.MinorNumber),
				Size:       directPVDrive.Status.TotalCapacity,
				Model:      directPVDrive.Status.ModelNumber,
				Vendor:     directPVDrive.Status.Vendor,
				Filesystem: "xfs",
				Status:     DeviceStatusOffline,
			})
		case 1:
			// Online drive detected
			if len(statuses) > 0 && !matcher.StringIn(statuses, string(DeviceStatusOnline)) {
				break
			}
			if len(drives) > 0 && !matcher.StringIn(drives, matchedDevices[0].Name) {
				break
			}
			deviceList = append(deviceList, Device{
				Name:       matchedDevices[0].Name,
				Major:      matchedDevices[0].Major,
				Minor:      matchedDevices[0].Minor,
				Size:       int64(matchedDevices[0].Size), // FixMe: Remove type conversion
				Model:      matchedDevices[0].Model,
				Vendor:     matchedDevices[0].Vendor,
				Filesystem: matchedDevices[0].FSType,
				Status:     DeviceStatusOnline,
				UDevData:   matchedDevices[0].UDevData,
			})
		default:
			// Multiple matches found for the Online drive
			klog.ErrorS(errDuplicateDevice, "drive: ", directPVDrive.Name, " devices: ", getDeviceNames(matchedDevices))
		}
		devices = unmatchedDevices
	}
	for _, device := range devices {
		deviceStatus := DeviceStatusAvailable
		if sys.IsDeviceUnavailable(device) {
			deviceStatus = DeviceStatusUnavailable
		}
		if len(statuses) > 0 && !matcher.StringIn(statuses, string(deviceStatus)) {
			continue
		}
		if len(drives) > 0 && !matcher.StringIn(drives, device.Name) {
			continue
		}
		deviceList = append(deviceList, Device{
			Name:       device.Name,
			Major:      device.Major,
			Minor:      device.Minor,
			Size:       int64(device.Size), // FixMe: Remove type conversion
			Model:      device.Model,
			Vendor:     device.Vendor,
			Filesystem: device.FSType,
			Status:     deviceStatus,
			UDevData:   device.UDevData,
		})
	}
	return deviceList, nil

}

func (n *nodeAPIHandler) listLocalDirectPVDrives(ctx context.Context) ([]directcsi.DirectCSIDrive, error) {
	labelSelector := fmt.Sprintf("%s=%s", utils.NodeLabelKey, utils.NewLabelValue(n.nodeID))
	result, err := client.GetLatestDirectCSIDriveInterface().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// formatLocalDrivesHandler formats the devices present in the node and returns back the status
func (n *nodeAPIHandler) formatLocalDrivesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("couldn't read the request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req FormatDevicesRequest
	if err = json.Unmarshal(data, &req); err != nil {
		klog.Errorf("couldn't parse the request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	formatDevices, ok := req.FormatInfo[NodeName(n.nodeID)]
	if !ok {
		klog.Errorf("nodename not found in the request: %s", n.nodeID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var formatStatusList []FormatDeviceStatus
	var wg sync.WaitGroup
	for _, formatDevice := range formatDevices {
		wg.Add(1)
		go func(device FormatDevice) {
			defer wg.Done()
			formatStatus := n.format(context.Background(), device)
			if formatStatus.Error == "" {
				// ToDo: create the DirectPVDrive object with name = formatStatus.FSUUID
				// if creation fails, umount the drive
			}
			formatStatusList = append(formatStatusList, formatStatus)
		}(formatDevice)
	}
	wg.Wait()
	// Marshal API response
	jsonBytes, err := json.Marshal(FormatDevicesResponse{
		DeviceInfo: map[NodeName][]FormatDeviceStatus{
			NodeName(n.nodeID): formatStatusList,
		},
	})
	if err != nil {
		klog.Errorf("Couldn't marshal the format status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (n *nodeAPIHandler) format(ctx context.Context, device FormatDevice) (formatStatus FormatDeviceStatus) {
	// Get format lock
	n.getFormatLock(device.Major, device.Minor).Lock()
	defer n.getFormatLock(device.Major, device.Minor).Unlock()
	formatStatus.Name = device.Name
	// Check if the udev data is matching
	udevData, err := n.readRunUdevDataByMajorMinor(device.Major, device.Minor)
	if err != nil {
		formatStatus.Error = err.Error()
		formatStatus.Reason = "couldn't read the udev data"
		klog.V(3).Infof("error while reading udevdata for device %s: %v", device.Name, err)
		return formatStatus
	}
	if !reflect.DeepEqual(udevData, device.UDevData) {
		formatStatus.Error = errUDevDataMismatch.Error()
		formatStatus.Suggestion = formatRetrySuggestion
		formatStatus.Reason = udevDataMismatchReason
		klog.V(3).Infof("udev data isn't matching for device %s", device.Name)
		return formatStatus
	}
	// Check if force is required
	if v, ok := udevData["ID_FS_TYPE"]; ok {
		if v != "" && !device.Force {
			formatStatus.Error = errForceRequired.Error()
			formatStatus.Reason = fmt.Sprintf("device %s already has a %s fs", device.Name, v)
			formatStatus.Suggestion = formatRetryWithForceSuggestion
			return formatStatus
		}
	}
	// Format the device
	fsuuid := uuid.New().String()
	err = n.makeFS(ctx, "/dev/"+device.Name, fsuuid, device.Force, n.reflinkSupport)
	if err != nil {
		formatStatus.Error = err.Error()
		formatStatus.Reason = fmt.Sprintf("failed to format device %s: %s", device.Name, err.Error())
		formatStatus.Suggestion = formatRetrySuggestion
		klog.Errorf("failed to format drive %s; %w", device.Name, err)
		return formatStatus
	}
	formatStatus.FSUUID = fsuuid
	// Mount the device
	mountTarget := path.Join(sys.MountRoot, fsuuid)
	err = n.mountDevice("/dev/"+device.Name, mountTarget, []string{})
	if err != nil {
		formatStatus.Error = err.Error()
		formatStatus.Reason = fmt.Sprintf("failed to mount device %s: %s", device.Name, err.Error())
		formatStatus.Suggestion = formatRetrySuggestion
		klog.Errorf("failed to mount drive %s; %w", device.Name, err)
		return formatStatus
	}
	// Umount the target on error
	defer func() {
		if formatStatus.Error != "" {
			if err := n.safeUnmount(mountTarget, false, false, false); err != nil {
				formatStatus.Error = err.Error()
				formatStatus.Reason = fmt.Sprintf("failed to umount %s: %v")
				formatStatus.Suggestion = fmt.Sprintf("please umount %s and retry the format request", mountTarget)
			}
		}
	}()
	// FIXME: probe fsinfo to calculate the allocatedcapacity
	// Write metadata
	if err := writeFormatMetadata(FormatMetadata{
		FSUUID:      fsuuid,
		FormattedBy: "v1beta1", // FixMe: Remove constants
	}, path.Join(mountTarget, ".directpv.sys", "metadata.json")); err != nil {
		klog.Errorf("failed to write metadata for device: %s: %s", device.Name, err.Error())
		formatStatus.Error = err.Error()
		formatStatus.Reason = fmt.Sprintf("failed to marshal device metadata for %s: %s", device.Name, err.Error())
		formatStatus.Suggestion = formatRetrySuggestion
		return
	}
	// Create symbolic link
	if err := os.Symlink(mountTarget, path.Join(mountTarget, fsuuid)); err != nil {
		klog.Errorf("failed to create symlink for target %s. device: %s err: %s", mountTarget, device.Name, err.Error())
		formatStatus.Error = err.Error()
		formatStatus.Reason = fmt.Sprintf("failed to create symlink for %s: %s", device.Name, err.Error())
		formatStatus.Suggestion = formatRetrySuggestion
	}
	return
}
