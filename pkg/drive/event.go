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
	"syscall"
	"time"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

const (
	workerThreads = 40
	resyncPeriod  = 10 * time.Minute
)

// SetIOError sets I/O error condition to specified drive.
func SetIOError(ctx context.Context, driveID directpvtypes.DriveID) error {
	updateFunc := func() error {
		drive, err := client.DriveClient().Get(ctx, string(driveID), metav1.GetOptions{})
		if err != nil {
			return err
		}
		client.Eventf(drive, client.EventTypeWarning, client.EventReasonDriveIOError, "I/O error occurred")
		drive.Status.Status = directpvtypes.DriveStatusError
		drive.SetIOErrorCondition()
		_, err = client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
		return err
	}
	if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
		return err
	}
	return nil
}

// StageVolume creates and mounts staging target path of the volume to the drive.
func StageVolume(
	ctx context.Context,
	volume *types.Volume,
	stagingTargetPath string,
	getDeviceByFSUUID func(fsuuid string) (string, error),
	mkdir func(volumeDir string) error,
	setQuota func(ctx context.Context, device, stagingTargetPath, volumeName string, quota xfs.Quota, update bool) error,
	bindMount func(volumeDir, stagingTargetPath string, readOnly bool) error,
) (codes.Code, error) {
	device, err := getDeviceByFSUUID(volume.Status.FSUUID)
	if err != nil {
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
		return codes.Internal, fmt.Errorf("unable to find device by FSUUID %v; %w", volume.Status.FSUUID, err)
	}

	volumeDir := types.GetVolumeDir(volume.Status.FSUUID, volume.Name)

	if err := mkdir(volumeDir); err != nil && !errors.Is(err, os.ErrExist) {
		if errors.Unwrap(err) == syscall.EIO {
			if err := SetIOError(ctx, volume.GetDriveID()); err != nil {
				return codes.Internal, fmt.Errorf("unable to set drive error; %w", err)
			}
		}
		klog.ErrorS(err, "unable to create volume directory", "VolumeDir", volumeDir)
		return codes.Internal, err
	}

	quota := xfs.Quota{
		HardLimit: uint64(volume.Status.TotalCapacity),
		SoftLimit: uint64(volume.Status.TotalCapacity),
	}

	if err := setQuota(ctx, device, volumeDir, volume.Name, quota, false); err != nil {
		klog.ErrorS(err, "unable to set quota on volume data path", "DataPath", volumeDir)
		return codes.Internal, fmt.Errorf("unable to set quota on volume data path; %w", err)
	}

	if stagingTargetPath != "" {
		if err := bindMount(volumeDir, stagingTargetPath, false); err != nil {
			return codes.Internal, fmt.Errorf("unable to bind mount volume directory to staging target path; %w", err)
		}
	}

	volume.Status.DataPath = volumeDir
	volume.Status.StagingTargetPath = stagingTargetPath
	volume.Status.Status = directpvtypes.VolumeStatusReady
	if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	}); err != nil {
		return codes.Internal, err
	}

	return codes.OK, nil
}

type driveEventHandler struct {
	nodeID            directpvtypes.NodeID
	getMounts         func() (mountPointMap, deviceMap map[string]utils.StringSet, err error)
	unmount           func(target string) error
	mkdir             func(path string) error
	bindMount         func(source, target string, readOnly bool) error
	getDeviceByFSUUID func(fsuuid string) (string, error)
	setQuota          func(ctx context.Context, device, path, volumeName string, quota xfs.Quota, update bool) (err error)
	rmdir             func(fsuuid string) error
}

func newDriveEventHandler(nodeID directpvtypes.NodeID) *driveEventHandler {
	return &driveEventHandler{
		nodeID: nodeID,
		getMounts: func() (mountPointMap, deviceMap map[string]utils.StringSet, err error) {
			mountPointMap, deviceMap, _, err = sys.GetMounts(false)
			return
		},
		unmount: func(mountPoint string) error {
			return sys.Unmount(mountPoint, true, true, false)
		},
		mkdir: func(dir string) error {
			return sys.Mkdir(dir, 0o755)
		},
		bindMount:         xfs.BindMount,
		getDeviceByFSUUID: sys.GetDeviceByFSUUID,
		setQuota:          xfs.SetQuota,
		rmdir: func(fsuuid string) (err error) {
			driveMountPoint := types.GetDriveMountDir(fsuuid)
			if err = os.Remove(driveMountPoint); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			driveMountPoint = path.Join("/var/lib/direct-csi/mnt", fsuuid)
			if err = os.Remove(driveMountPoint); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		},
	}
}

func (handler *driveEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, handler.nodeID)
	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.DriveResource,
		"",
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		},
	)
}

func (handler *driveEventHandler) ObjectType() runtime.Object {
	return &types.Drive{}
}

func (handler *driveEventHandler) unmountDrive(ctx context.Context, drive *types.Drive, skipDriveMount bool) error {
	mountPointMap, deviceMap, err := handler.getMounts()
	if err != nil {
		return err
	}
	driveMountPoint := types.GetDriveMountDir(drive.Status.FSUUID)
	devices, found := mountPointMap[driveMountPoint]
	if !found {
		// Check for legacy mount for backward compatibility.
		driveMountPoint = path.Join("/var/lib/direct-csi/mnt", drive.Status.FSUUID)
		if devices, found = mountPointMap[driveMountPoint]; !found {
			return nil // Device already umounted
		}
	}
	if len(devices) > 1 {
		return fmt.Errorf("multiple devices %v are mounted for FSUUID %v", devices, drive.Status.FSUUID)
	}
	mountpoints := deviceMap[devices.ToSlice()[0]]
	for _, mountPoint := range mountpoints.ToSlice() {
		if skipDriveMount && mountPoint == driveMountPoint {
			continue
		}

		if err := handler.unmount(mountPoint); err != nil {
			return err
		}
	}
	return nil
}

func (handler *driveEventHandler) remove(ctx context.Context, drive *types.Drive) error {
	volumeCount := drive.GetVolumeCount()
	if volumeCount > 0 {
		return fmt.Errorf("drive %v still contains %v volumes", drive.GetDriveID(), volumeCount)
	}
	if err := handler.unmountDrive(ctx, drive, false); err != nil {
		return err
	}
	drive.RemoveFinalizers()
	if _, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()}); err != nil {
		return err
	}
	if err := client.DriveClient().Delete(ctx, drive.Name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := handler.rmdir(drive.Status.FSUUID); err != nil {
		klog.ErrorS(err, "unable to remove drive mount directory", "drive", drive.GetDriveID())
	}

	return nil
}

func (handler *driveEventHandler) move(ctx context.Context, drive *types.Drive) error {
	for _, volumeName := range drive.GetVolumes() {
		volume, err := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{})
		if err != nil {
			klog.ErrorS(err, "unable to retrieve volume", "volume", volume.Name)
			return err
		}

		if volume.Status.FSUUID == drive.Status.FSUUID {
			continue
		}

		if volume.IsPublished() {
			return fmt.Errorf("cannot move published volume %v to drive ID %v", volume.Name, drive.GetDriveID())
		}

		if volume.GetNodeID() != drive.GetNodeID() {
			return fmt.Errorf(
				"volume %v must be on same node of destination drive; volume node %v; desination node %v",
				volume.Name,
				volume.GetNodeID(),
				drive.GetNodeID(),
			)
		}

		srcDriveID := volume.GetDriveID()
		volume.Status.FSUUID = drive.Status.FSUUID
		volume.SetDriveID(drive.GetDriveID())
		volume.SetDriveName(drive.GetDriveName())
		volume.Status.DataPath = ""
		if volume.IsStaged() {
			_, err = StageVolume(
				ctx,
				volume,
				volume.Status.StagingTargetPath,
				handler.getDeviceByFSUUID,
				handler.mkdir,
				handler.setQuota,
				handler.bindMount,
			)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				klog.ErrorS(err, "unable to stage volume after volume move",
					"volume", volume.Name,
					"dataPath", volume.Status.DataPath,
					"stagingTargetPath", volume.Status.StagingTargetPath,
				)
			}
		} else {
			volume.Status.Status = directpvtypes.VolumeStatusPending
		}

		if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		}); err != nil {
			return err
		}

		client.Eventf(
			volume, client.EventTypeNormal, client.EventReasonVolumeMoved,
			"Volume moved from drive %v to drive %v", srcDriveID, volume.GetDriveID(),
		)
	}

	drive.Status.Status = directpvtypes.DriveStatusReady
	_, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{
		TypeMeta: types.NewDriveTypeMeta(),
	})
	return err
}

func (handler *driveEventHandler) handleUpdate(ctx context.Context, drive *types.Drive) error {
	switch drive.Status.Status {
	case directpvtypes.DriveStatusRemoved:
		return handler.remove(ctx, drive)
	case directpvtypes.DriveStatusMoving:
		return handler.move(ctx, drive)
	}

	return nil
}

func (handler *driveEventHandler) Handle(ctx context.Context, eventType controller.EventType, object runtime.Object) error {
	switch eventType {
	case controller.AddEvent, controller.UpdateEvent:
		return handler.handleUpdate(ctx, object.(*types.Drive))
	}

	return nil
}

// StartController starts drive controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID) {
	ctrl := controller.New(ctx, "drive", newDriveEventHandler(nodeID), workerThreads, resyncPeriod)
	ctrl.Run(ctx)
}
