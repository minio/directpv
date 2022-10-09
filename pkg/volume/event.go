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
	"os"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type volumeEventHandler struct {
	nodeID            directpvtypes.NodeID
	unmount           func(target string) error
	getDeviceByFSUUID func(fsuuid string) (string, error)
	mkdir             func(path string) error
	setQuota          func(ctx context.Context, device, path, volumeName string, quota xfs.Quota) (err error)
	bindMount         func(source, target string, readOnly bool) error
}

func newVolumeEventHandler(nodeID directpvtypes.NodeID) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID: nodeID,
		unmount: func(mountPoint string) error {
			return sys.Unmount(mountPoint, true, true, false)
		},
		getDeviceByFSUUID: sys.GetDeviceByFSUUID,
		mkdir: func(dir string) error {
			return os.Mkdir(dir, 0o755)
		},
		setQuota:  xfs.SetQuota,
		bindMount: xfs.BindMount,
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

func (handler *volumeEventHandler) Name() string {
	return "volume"
}

func (handler *volumeEventHandler) ObjectType() runtime.Object {
	return &types.Volume{}
}

func (handler *volumeEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	volume, err := client.VolumeClient().Get(
		ctx, args.Object.(*types.Volume).Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)
	if err != nil {
		return err
	}

	if !volume.GetDeletionTimestamp().IsZero() {
		return handler.delete(ctx, volume)
	}

	return nil
}

func (handler *volumeEventHandler) delete(ctx context.Context, volume *types.Volume) error {
	if !volume.IsReleased() {
		return fmt.Errorf("volume %v must be released before cleaning up", volume.Name)
	}

	if volume.Status.TargetPath != "" {
		if err := handler.unmount(volume.Status.TargetPath); err != nil {
			if _, ok := err.(*os.PathError); !ok {
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
			if _, ok := err.(*os.PathError); !ok {
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
	if err := handler.releaseVolume(ctx, volume.GetDriveID(), volume.Name, volume.Status.TotalCapacity); err != nil {
		return err
	}

	volume.RemovePurgeProtection()
	_, err := client.VolumeClient().Update(
		ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)

	return err
}

func (handler *volumeEventHandler) releaseVolume(ctx context.Context, driveID directpvtypes.DriveID, volumeName string, capacity int64) error {
	drive, err := client.DriveClient().Get(
		ctx, string(driveID), metav1.GetOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		return err
	}

	found := drive.RemoveVolumeFinalizer(volumeName)
	if found {
		drive.Status.FreeCapacity += capacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity

		_, err = client.DriveClient().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
	}

	return err
}

// StartController starts volume controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID) error {
	listener := listener.NewListener(newVolumeEventHandler(nodeID), "volume-controller", string(nodeID), 40)
	return listener.Run(ctx)
}
