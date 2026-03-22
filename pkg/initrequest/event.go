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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/controller"
	pkgdevice "github.com/minio/directpv/pkg/device"
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
	workerThreads = 10
	resyncPeriod  = 5 * time.Minute
)

type initRequestEventHandler struct {
	nodeID   directpvtypes.NodeID
	reflink  bool
	topology map[string]string

	probeDevices func() ([]pkgdevice.Device, error)
	getDevices   func(majorMinor ...string) ([]pkgdevice.Device, error)
	getMounts    func() (*sys.MountInfo, error)
	makeFS       func(device, fsuuid string, force, reflink bool) (string, string, uint64, uint64, error)
	mount        func(device, fsuuid string) error
	unmount      func(fsuuid string) error
	symlink      func(fsuuid string) error
	makeMetaDir  func(fsuuid string) error
	writeFile    func(fsuuid, data string) error

	mu sync.Mutex
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
		getMounts: func() (mountInfo *sys.MountInfo, err error) {
			if mountInfo, err = sys.NewMountInfo(); err != nil {
				err = fmt.Errorf("unable get mount info from /proc; %w", err)
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

func (handler *initRequestEventHandler) ObjectType() runtime.Object {
	return &types.InitRequest{}
}

func (handler *initRequestEventHandler) Handle(ctx context.Context, eventType controller.EventType, object runtime.Object) error {
	switch eventType {
	case controller.UpdateEvent, controller.AddEvent:
		initRequest := object.(*types.InitRequest)
		if initRequest.Status.Status == directpvtypes.InitStatusPending {
			return handler.initDevices(ctx, initRequest)
		}
	default:
	}
	return nil
}

func (handler *initRequestEventHandler) initDevices(ctx context.Context, req *types.InitRequest) error {
	handler.mu.Lock()
	defer handler.mu.Unlock()

	var majorMinorList []string
	for i := range req.Spec.Devices {
		tokens := strings.SplitN(req.Spec.Devices[i].ID, "$", 2)
		if len(tokens) != 2 {
			client.Eventf(req, client.EventTypeWarning, client.EventReasonInitError, "invalid device ID %v", req.Spec.Devices[i])
			return updateInitRequest(ctx, req.Name, req.Status.Results, directpvtypes.InitStatusError)
		}
		majorMinorList = append(majorMinorList, tokens[0])
	}

	devices, err := handler.getDevices(majorMinorList...)
	if err != nil {
		client.Eventf(req, client.EventTypeWarning, client.EventReasonInitError, "probing failed with %s", err.Error())
		return updateInitRequest(ctx, req.Name, req.Status.Results, directpvtypes.InitStatusError)
	}
	probedDevices := map[string]pkgdevice.Device{}
	for _, device := range devices {
		probedDevices[device.MajorMinor] = device
	}

	results := make([]types.InitDeviceResult, len(req.Spec.Devices))
	var wg sync.WaitGroup
	for i := range req.Spec.Devices {
		results[i].Name = req.Spec.Devices[i].Name
		majorMinor := strings.SplitN(req.Spec.Devices[i].ID, "$", 2)[0]
		device, found := probedDevices[majorMinor]
		switch {
		case !found:
			results[i].Error = "device not found"
		case device.ID(handler.nodeID) != req.Spec.Devices[i].ID:
			results[i].Error = "device state changed"
		default:
			if deniedReason := device.DeniedReason(); deniedReason == "" {
				wg.Add(1)
				go func(i int, device pkgdevice.Device, force bool) {
					defer wg.Done()
					if err := handler.initDevice(device, force); err != nil {
						results[i].Error = err.Error()
					}
				}(i, device, req.Spec.Devices[i].Force || device.PartTableType() != "")
			} else {
				results[i].Error = "device init not permitted; " + deniedReason
			}
		}
	}
	wg.Wait()

	return updateInitRequest(ctx, req.Name, results, directpvtypes.InitStatusProcessed)
}

func updateInitRequest(ctx context.Context, name string, results []types.InitDeviceResult, status directpvtypes.InitStatus) error {
	updateFunc := func() error {
		initRequest, err := client.InitRequestClient().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		initRequest.Status.Results = results
		initRequest.Status.Status = status
		if _, err := client.InitRequestClient().Update(ctx, initRequest, metav1.UpdateOptions{TypeMeta: types.NewInitRequestTypeMeta()}); err != nil {
			return err
		}
		return nil
	}
	return retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
}

func (handler *initRequestEventHandler) initDevice(device pkgdevice.Device, force bool) error {
	devPath := utils.AddDevPrefix(device.Name)

	mountInfo, err := handler.getMounts()
	if err != nil {
		return err
	}

	mountPoints := make(utils.StringSet)
	for _, mountEntry := range mountInfo.FilterByMajorMinor(device.MajorMinor).List() {
		mountPoints.Set(mountEntry.MountPoint)
	}

	if len(mountPoints) != 0 {
		return fmt.Errorf("device %v mounted at %v", devPath, mountPoints.ToSlice())
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
			err = errors.Join(err, uerr)
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

// StartController starts initrequest controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID, identity, rack, zone, region string) {
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
		klog.ErrorS(err, "unable to create initrequest event handler")
		return
	}
	ctrl := controller.New("initrequest", initRequestHandler, workerThreads, resyncPeriod)
	ctrl.Run(ctx)
}
