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

package drive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	errDriveInUse = errors.New("drive still has volumes in-use")
)

type driveEventHandler struct {
	nodeID      string
	getMounts   func() (mountPointMap, deviceMap map[string][]string, err error)
	unmount     func(target string, force, detach, expire bool) error
	safeUnmount func(target string, force, detach, expire bool) error
	stageVolume func(ctx context.Context, volume *types.Volume) error
}

func newDriveEventHandler(nodeID string) *driveEventHandler {
	return &driveEventHandler{
		nodeID:      nodeID,
		getMounts:   sys.GetMounts,
		unmount:     sys.Unmount,
		safeUnmount: sys.SafeUnmount,
		stageVolume: volume.Stage,
	}
}

func (handler *driveEventHandler) ListerWatcher() cache.ListerWatcher {
	return DrivesListerWatcher(handler.nodeID)
}

func (handler *driveEventHandler) Name() string {
	return "drive"
}

func (handler *driveEventHandler) ObjectType() runtime.Object {
	return &types.Drive{}
}

func (handler *driveEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.AddEvent, listener.UpdateEvent:
		return handler.handleUpdate(ctx, args.Object.(*types.Drive))
	case listener.DeleteEvent:
	}
	return nil
}

func (handler *driveEventHandler) handleUpdate(ctx context.Context, drive *types.Drive) error {
	switch drive.Status.Status {
	case directpvtypes.DriveStatusReleased:
		return handler.release(ctx, drive)
	case directpvtypes.DriveStatusMoving:
		return handler.move(ctx, drive)
	default:
		return nil
	}
}

// move processes the move request by staging the moved target volumes
func (handler *driveEventHandler) move(ctx context.Context, drive *types.Drive) error {
	driveClient := client.DriveClient()
	volumeClient := client.VolumeClient()
	finalizers := drive.GetFinalizers()
	sourceDriveName := ""
	for _, finalizer := range finalizers {
		if finalizer == consts.DriveFinalizerDataProtection {
			continue
		}
		volumeName := strings.TrimPrefix(finalizer, consts.DriveFinalizerPrefix)
		volume, err := volumeClient.Get(ctx, volumeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("unable to retrieve volume %s: %v", volume.Name, err)
			return err
		}
		if sourceDriveName == "" {
			sourceDriveName = volume.Status.DriveName
		}
		if volume.IsPublished() {
			klog.Errorf("drive still has published volume: %s", volume.Name)
			return errDriveInUse
		}
		if err := handler.unmountVolume(ctx, volume); err != nil {
			klog.Errorf("unable to umount volume: %s: %v", volume.Name, err)
			return err
		}
		volume.Status.FSUUID = drive.Status.FSUUID
		volume.Status.DriveName = drive.Name
		if volume.IsStaged() {
			// Stage the volume if the volume in the source is staged already
			volume.Status.DataPath = types.GetVolumeDir(volume.Status.FSUUID, volume.Name)
			if err := handler.stageVolume(ctx, volume); err != nil {
				klog.ErrorS(err, "unable to stage volume",
					"volume", volume.Name,
					"dataPath", volume.Status.DataPath,
					"stagingTargetPath", volume.Status.StagingTargetPath,
				)
				return err
			}
		}
		// update the volume
		types.UpdateLabels(volume, map[types.LabelKey]types.LabelValue{
			types.DrivePathLabelKey: types.NewLabelValue(utils.TrimDevPrefix(drive.Status.Path)),
			types.DriveLabelKey:     types.NewLabelValue(drive.Name),
		})
		if _, err := volumeClient.Update(ctx, volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		}); err != nil {
			return err
		}
	}
	// update the source drive's status as "Moved"
	sourceDrive, err := driveClient.Get(ctx, sourceDriveName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to get the source drive",
			"sourcerive", sourceDrive.Name,
		)
		return err
	}
	sourceDrive.Status.Status = directpvtypes.DriveStatusMoved
	if _, err := driveClient.Update(ctx, sourceDrive, metav1.UpdateOptions{
		TypeMeta: types.NewDriveTypeMeta(),
	}); err != nil {
		return err
	}
	// Revert the status back to "Cordoned" as the transfer is successful
	drive.Status.Status = directpvtypes.DriveStatusCordoned
	_, err = driveClient.Update(ctx, drive, metav1.UpdateOptions{
		TypeMeta: types.NewDriveTypeMeta(),
	})
	return err
}

func (handler *driveEventHandler) unmountVolume(ctx context.Context, volume *types.Volume) error {
	if volume.Status.TargetPath != "" {
		if err := handler.safeUnmount(volume.Status.TargetPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.ErrorS(err, "unable to unmount container path",
					"volume", volume.Name,
					"targetPath", volume.Status.TargetPath,
				)
				return err
			}
		}
	}
	if volume.Status.StagingTargetPath != "" {
		if err := handler.safeUnmount(volume.Status.StagingTargetPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.ErrorS(err, "unable to unmount staging path",
					"volume", volume.Name,
					"StagingTargetPath", volume.Status.StagingTargetPath,
				)
				return err
			}
		}
	}
	return nil
}

func (handler *driveEventHandler) release(ctx context.Context, drive *types.Drive) error {
	finalizers := drive.GetFinalizers()
	if len(finalizers) > 1 {
		return fmt.Errorf("unable to release drive %s. the drive still has volumes to be cleaned up", drive.Name)
	}
	if err := handler.unmountDrive(ctx, drive); err != nil {
		return err
	}
	drive.Finalizers, _ = utils.ExcludeFinalizer(
		finalizers, consts.DriveFinalizerDataProtection,
	)
	if _, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()}); err != nil {
		return err
	}
	return client.DriveClient().Delete(ctx, drive.Name, metav1.DeleteOptions{})
}

func (handler *driveEventHandler) unmountDrive(ctx context.Context, drive *types.Drive) error {
	mountPointMap, deviceMap, err := handler.getMounts()
	if err != nil {
		return err
	}
	devices, ok := mountPointMap[path.Join(consts.MountRootDir, drive.Status.FSUUID)]
	if !ok {
		devices, ok = mountPointMap[path.Join(consts.LegacyMountRootDir, drive.Status.FSUUID)]
		if !ok {
			// Device umounted already
			return nil
		}
	}
	if len(devices) > 1 {
		return fmt.Errorf("drive %s mounted is mounted in more than one place", drive.Name)
	}
	mountpoints := deviceMap[devices[0]]
	for _, mountPoint := range mountpoints {
		if err := handler.unmount(mountPoint, true, true, false); err != nil {
			return err
		}
	}
	return nil
}

// StartController starts drive controller.
func StartController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(newDriveEventHandler(nodeID), "drive-controller", hostname, 40)
	return listener.Run(ctx)
}
