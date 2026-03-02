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

package volume

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	workerThreads = 40
	resyncPeriod  = 10 * time.Minute
)

type volumeEventHandler struct {
	nodeID            directpvtypes.NodeID
	unmount           func(target string) error
	getDeviceByFSUUID func(fsuuid string) (string, error)
	removeQuota       func(ctx context.Context, device, path, volumeName string) error
}

func newVolumeEventHandler(nodeID directpvtypes.NodeID) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID: nodeID,
		unmount: func(mountPoint string) error {
			return sys.Unmount(mountPoint, true, true, false)
		},
		getDeviceByFSUUID: sys.GetDeviceByFSUUID,
		removeQuota: func(ctx context.Context, device, path, volumeName string) error {
			return xfs.SetQuota(ctx, device, path, volumeName, xfs.Quota{}, true)
		},
	}
}

func (handler *volumeEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, handler.nodeID)
	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.VolumeResource,
		"",
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		},
	)
}

func (handler *volumeEventHandler) ObjectType() runtime.Object {
	return &types.Volume{}
}

func (handler *volumeEventHandler) Handle(ctx context.Context, eventType controller.EventType, object runtime.Object) error {
	volume := object.(*types.Volume)
	if !volume.GetDeletionTimestamp().IsZero() {
		return handler.delete(ctx, volume)
	}

	if eventType == controller.AddEvent {
		return sync(ctx, volume)
	}

	return nil
}

func sync(ctx context.Context, volume *types.Volume) error {
	drive, err := client.DriveClient().Get(ctx, string(volume.GetDriveID()), metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}
	driveName := drive.GetDriveName()
	if volume.GetDriveName() != driveName {
		volume.SetDriveName(driveName)
		_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		})
	}
	return err
}

func (handler *volumeEventHandler) delete(ctx context.Context, volume *types.Volume) error {
	volume, err := client.VolumeClient().Get(ctx, volume.GetName(), metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !volume.IsReleased() {
		return fmt.Errorf("volume %v must be released before cleaning up", volume.Name)
	}

	if volume.Status.TargetPath != "" {
		if err := handler.unmount(volume.Status.TargetPath); err != nil {
			var perr *fs.PathError
			if !errors.As(err, &perr) {
				klog.ErrorS(err, "unable to unmount container path",
					"volume", volume.Name,
					"containerPath", volume.Status.TargetPath,
				)
				return err
			}
		}
	}
	if volume.Status.StagingTargetPath != "" {
		if err := handler.unmount(volume.Status.StagingTargetPath); err != nil {
			var perr *fs.PathError
			if !errors.As(err, &perr) {
				klog.ErrorS(err, "unable to unmount staging path",
					"volume", volume.Name,
					"StagingTargetPath", volume.Status.StagingTargetPath,
				)
				return err
			}
		}
	}

	deletedDir := volume.Status.DataPath + ".deleted"
	if err := os.Rename(volume.Status.DataPath, deletedDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		// FIXME: Also handle input/output error
		klog.ErrorS(
			err,
			"unable to rename data path to deleted data path",
			"volume", volume.Name,
			"DataPath", volume.Status.DataPath,
			"DeletedDir", deletedDir,
		)
		return err
	}

	go func(volumeName, deletedDir string) {
		if err := os.RemoveAll(deletedDir); err != nil {
			klog.ErrorS(
				err,
				"unable to remove deleted data path",
				"volume", volumeName,
				"DeletedDir", deletedDir,
			)
		}
	}(volume.Name, deletedDir)

	// Release volume from associated drive.
	if err := handler.releaseVolume(ctx, volume); err != nil {
		return err
	}

	volume.RemovePurgeProtection()
	_, err = client.VolumeClient().Update(
		ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)

	if err == nil {
		if len(volume.Finalizers) != 0 { // This should not happen here.
			client.Eventf(volume, client.EventTypeNormal, client.EventReasonVolumeReleased, "volume is released")
		}
	}

	return err
}

func (handler *volumeEventHandler) releaseVolume(ctx context.Context, volume *types.Volume) error {
	drive, err := client.DriveClient().Get(
		ctx, string(volume.GetDriveID()), metav1.GetOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		return err
	}

	found := drive.RemoveVolumeFinalizer(volume.Name)
	if found {
		if device, err := handler.getDeviceByFSUUID(volume.Status.FSUUID); err != nil {
			klog.ErrorS(
				err,
				"unable to find device by FSUUID; "+
					"either device is removed or run command "+
					"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
					"on the host to reload",
				"FSUUID", volume.Status.FSUUID)
			client.Eventf(
				volume, client.EventTypeWarning, client.EventReasonStageVolume,
				"unable to find device by FSUUID %v; "+
					"either device is removed or run command "+
					"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
					" on the host to reload", volume.Status.FSUUID)
		} else if err := handler.removeQuota(ctx, device, volume.Status.DataPath, volume.Name); err != nil {
			klog.ErrorS(err, "unable to remove quota on volume data path", "DataPath", volume.Status.DataPath)
		}

		drive.Status.FreeCapacity += volume.Status.TotalCapacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity
		drive.RemoveVolumeClaimID(volume.GetClaimID())
		_, err = client.DriveClient().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
	}

	return err
}

// StartController starts volume controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID) {
	ctrl := controller.New("volume", newVolumeEventHandler(nodeID), workerThreads, resyncPeriod)
	ctrl.Run(ctx)
}
