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

package initrequest

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	pkgdevice "github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

const (
	initReqSpan = time.Minute * 5
)

type initRequestEventHandler struct {
	nodeID   directpvtypes.NodeID
	reflink  bool
	topology map[string]string

	probeDevices func() ([]pkgdevice.Device, error)
	getDevices   func(majorMinor ...string) ([]pkgdevice.Device, error)
	getMounts    func() (map[string]utils.StringSet, map[string]utils.StringSet, error)
	makeFS       func(device, fsuuid string, force, reflink bool) (string, string, uint64, uint64, error)
	mount        func(device, fsuuid string) error
	unmount      func(fsuuid string) error
	symlink      func(fsuuid string) error
	makeMetaDir  func(fsuuid string) error
	writeFile    func(fsuuid, data string) error
}

func newInitRequestEventHandler(ctx context.Context, nodeID directpvtypes.NodeID, topology map[string]string) (*initRequestEventHandler, error) {
	reflink, err := reflinkSupported(ctx)
	if err != nil {
		return nil, err
	}

	if reflink {
		klog.V(3).Infof("XFS reflink support is enabled")
	} else {
		klog.V(3).Infof("XFS reflink support is disabled")
	}

	return &initRequestEventHandler{
		reflink:  reflink,
		nodeID:   nodeID,
		topology: topology,

		probeDevices: pkgdevice.Probe,
		getDevices:   pkgdevice.ProbeDevices,
		getMounts: func() (deviceMap, majorMinorMap map[string]utils.StringSet, err error) {
			if _, deviceMap, majorMinorMap, err = sys.GetMounts(true); err != nil {
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
		makeMetaDir: func(fsuuid string) (err error) {
			if err = sys.Mkdir(types.GetDriveMetaDir(fsuuid), 0o750); err != nil {
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

func (handler *initRequestEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, handler.nodeID)
	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.InitRequestResource,
		"",
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		},
	)
}

func (handler *initRequestEventHandler) Name() string {
	return "initrequest"
}

func (handler *initRequestEventHandler) ObjectType() runtime.Object {
	return &types.InitRequest{}
}

func (handler *initRequestEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.UpdateEvent, listener.SyncEvent:
		initRequest := args.Object.(*types.InitRequest)
		if time.Since(initRequest.CreationTimestamp.Time) >= initReqSpan {
			// Delete the initrequest if it exceeds the span
			return client.InitRequestClient().Delete(ctx, initRequest.Name, metav1.DeleteOptions{})
		}
		if initRequest.Status.Status == directpvtypes.InitStatusPending {
			return handler.initDevices(ctx, initRequest)
		}
	case listener.AddEvent, listener.DeleteEvent:
	}
	return nil
}

func (handler *initRequestEventHandler) initDevices(ctx context.Context, req *types.InitRequest) error {
	var majorMinorList []string
	for i := range req.Spec.Devices {
		majorMinorList = append(majorMinorList, req.Spec.Devices[i].MajorMinor)
	}

	devices, err := handler.getDevices(majorMinorList...)
	if err != nil {
		return err
	}
	probedDevices := map[string]pkgdevice.Device{}
	for _, device := range devices {
		probedDevices[device.MajorMinor] = device
	}

	results := make([]types.InitDeviceResult, len(req.Spec.Devices))
	var wg sync.WaitGroup
	var failed bool
	for i := range req.Spec.Devices {
		results[i].Name = req.Spec.Devices[i].Name
		device, found := probedDevices[req.Spec.Devices[i].MajorMinor]
		switch {
		case !found:
			failed = true
			results[i].Error = "device not found"
		case device.ID(handler.nodeID) != req.Spec.Devices[i].ID:
			failed = true
			results[i].Error = "device's state changed"
		default:
			wg.Add(1)
			go func(i int, device pkgdevice.Device, force bool) {
				defer wg.Done()
				if err := handler.init(ctx, device, force); err != nil {
					failed = true
					results[i].Error = err.Error()
				}
			}(i, device, req.Spec.Devices[i].Force)
		}
	}
	wg.Wait()

	updateFunc := func() error {
		initRequest, err := client.InitRequestClient().Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		initRequest.Status.Results = results
		initRequest.Status.Status = directpvtypes.InitStatusSuccess
		if failed {
			initRequest.Status.Status = directpvtypes.InitStatusFailed
		}
		if _, err := client.InitRequestClient().Update(ctx, initRequest, metav1.UpdateOptions{TypeMeta: types.NewInitRequestTypeMeta()}); err != nil {
			return err
		}
		return nil
	}
	if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
		return err
	}
	return nil
}

func (handler *initRequestEventHandler) init(ctx context.Context, device pkgdevice.Device, force bool) error {
	devPath := utils.AddDevPrefix(device.Name)

	deviceMap, majorMinorMap, err := handler.getMounts()
	if err != nil {
		return err
	}

	var mountPoints []string
	if devices, found := majorMinorMap[device.MajorMinor]; found {
		for _, name := range devices.ToSlice() {
			mountPoints = append(mountPoints, deviceMap[name].ToSlice()...)
		}
	}
	if len(mountPoints) != 0 {
		return fmt.Errorf("device %v mounted at %v", devPath, mountPoints)
	}

	fsuuid := uuid.New().String()

	_, _, totalCapacity, freeCapacity, err := handler.makeFS(devPath, fsuuid, force, handler.reflink)
	if err != nil {
		return err
	}

	if err = handler.mount(devPath, fsuuid); err != nil {
		return err
	}
	defer func() {
		if err == nil {
			return
		}
		if uerr := handler.unmount(fsuuid); uerr != nil {
			err = fmt.Errorf("%w; %v", err, uerr)
		}
	}()

	if err = handler.symlink(fsuuid); err != nil {
		return err
	}

	if err = handler.makeMetaDir(fsuuid); err != nil {
		return err
	}

	data := fmt.Sprintf("APP_NAME=%v\nAPP_VERSION=%v\nFSUUID=%v\n", consts.AppName, consts.LatestAPIVersion, fsuuid)
	if err = handler.writeFile(fsuuid, data); err != nil {
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
			Topology:      handler.topology,
		},
		handler.nodeID,
		directpvtypes.DriveName(device.Name),
		directpvtypes.AccessTierDefault,
	)
	if _, err = client.DriveClient().Create(context.Background(), drive, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("unable to create Drive CRD; %w", err)
	}
	return nil
}

// StartController starts node controller.
func StartController(ctx context.Context, identity string, nodeID directpvtypes.NodeID, rack, zone, region string) error {
	initRequestHandler, err := newInitRequestEventHandler(
		ctx,
		nodeID,
		map[string]string{
			string(directpvtypes.TopologyDriverIdentity): identity,
			string(directpvtypes.TopologyDriverRack):     rack,
			string(directpvtypes.TopologyDriverZone):     zone,
			string(directpvtypes.TopologyDriverRegion):   region,
			string(directpvtypes.TopologyDriverNode):     string(nodeID),
		},
	)
	if err != nil {
		return err
	}
	listener := listener.NewListener(initRequestHandler, "initrequest-controller", string(nodeID), 40)
	return listener.Run(ctx)
}