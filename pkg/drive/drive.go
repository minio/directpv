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
	"path/filepath"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/fs/xfs"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

var (
	errNotMounted          = errors.New("drive not mounted")
	errInvalidMountOptions = errors.New("invalid mount options")
)

type driveEventHandler struct {
	nodeID          string
	reflinkSupport  bool
	getDevice       func(major, minor uint32) (string, error)
	stat            func(name string) (os.FileInfo, error)
	mountDevice     func(fsUUID, target string, flags []string) error
	unmountDevice   func(device string) error
	makeFS          func(ctx context.Context, device, uuid string, force, reflink bool) error
	getFreeCapacity func(path string) (uint64, error)
}

// StartController starts drive event controller.
func StartController(ctx context.Context, nodeID string, reflinkSupport bool) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(
		newDriveEventHandler(nodeID, reflinkSupport),
		"drive-controller",
		hostname,
		40,
	)
	return listener.Run(ctx)
}

func newDriveEventHandler(nodeID string, reflinkSupport bool) *driveEventHandler {
	return &driveEventHandler{
		nodeID:         nodeID,
		reflinkSupport: reflinkSupport,
		getDevice:      getDevice,
		stat:           os.Stat,
		mountDevice:    mount.MountXFSDevice,
		unmountDevice:  mount.UnmountDevice,
		makeFS:         xfs.MakeFS,
	}
}

func (handler *driveEventHandler) ListerWatcher() cache.ListerWatcher {
	return client.DrivesListerWatcher(handler.nodeID)
}

func (handler *driveEventHandler) KubeClient() kubernetes.Interface {
	return client.GetKubeClient()
}

func (handler *driveEventHandler) Name() string {
	return "drive"
}

func (handler *driveEventHandler) ObjectType() runtime.Object {
	return &directcsi.DirectCSIDrive{}
}

func (handler *driveEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.AddEvent:
		return handler.add(ctx, args.Object.(*directcsi.DirectCSIDrive))
	case listener.UpdateEvent:
		return handler.update(ctx, args.Object.(*directcsi.DirectCSIDrive))
	case listener.DeleteEvent:
		return handler.delete(ctx, args.Object.(*directcsi.DirectCSIDrive))
	}
	return nil
}

func (handler *driveEventHandler) add(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	err := verifyHostStateForDrive(drive)
	switch err {
	case nil:
		switch {
		case isFormatRequested(drive):
			return handler.format(ctx, drive)
		case drive.Status.DriveStatus == directcsi.DriveStatusReleased:
			return handler.release(ctx, drive)
		default:
			return nil
		}
	case errNotMounted:
		return handler.mountDrive(ctx, drive, false)
	case errInvalidMountOptions:
		return handler.mountDrive(ctx, drive, true)
	case os.ErrNotExist:
		return handler.lost(ctx, drive)
	default:
		return err
	}

}

func (handler *driveEventHandler) update(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	err := verifyHostStateForDrive(drive)
	switch err {
	case nil:
		switch {
		case isFormatRequested(drive):
			return handler.format(ctx, drive)
		case drive.Status.DriveStatus == directcsi.DriveStatusReleased:
			return handler.release(ctx, drive)
		default:
			return nil
		}
	case os.ErrNotExist:
		return handler.lost(ctx, drive)
	default:
		return err
	}
}

func (handler *driveEventHandler) delete(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	return nil
}

func (handler *driveEventHandler) format(ctx context.Context, drive *directcsi.DirectCSIDrive) (err error) {
	device, err := handler.getDevice(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		klog.Error(err)
		return err
	}

	// stat check
	if _, err := handler.stat(device); err != nil {
		err = fmt.Errorf("unable to read device %v of major/minor %v:%v; %w", device, drive.Status.MajorNumber, drive.Status.MinorNumber, err)
		klog.Error(err)
		return err
	}

	// validate request
	if drive.Status.Filesystem != "" && !drive.Spec.RequestedFormat.Force {
		err = fmt.Errorf("drive already has an FS %s. Please set `--force`", drive.Status.Filesystem)
	}

	filesystemUUID := getFSUUIDFromDrive(drive)
	// umount the drive if mounted
	if err == nil {
		err = handler.unmountDevice(device)
		if err != nil {
			klog.Errorf("failed to umount drive %s; %w", drive.Name, err)
		}
	}
	// format the drive
	if err == nil {
		err = handler.makeFS(ctx, drive.Status.Path, filesystemUUID, drive.Spec.RequestedFormat.Force, handler.reflinkSupport)
		if err != nil {
			klog.Errorf("failed to format drive %s; %w", drive.Name, err)
		}
	}
	// mount the drive
	if err == nil {
		target := filepath.Join(sys.MountRoot, drive.Name)
		err = handler.mountDevice(device, target, drive.Spec.RequestedFormat.MountOptions)
		if err != nil {
			klog.Errorf("failed to mount drive %s; %w", drive.Name, err)
		}
	}

	if err != nil {
		utils.UpdateCondition(
			drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionOwned),
			metav1.ConditionFalse,
			string(directcsi.DirectCSIDriveReasonAdded),
			err.Error(),
		)
		_, uerr := client.GetLatestDirectCSIDriveInterface().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
		)
		if uerr != nil && err == nil {
			err = uerr
		}
	}

	return err
}

func (handler *driveEventHandler) release(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	device, err := handler.getDevice(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		klog.Error(err)
		return err
	}
	err = handler.unmountDevice(device)
	if err != nil {
		klog.Errorf("failed to release drive %s; %w", drive.Name, err)
		utils.UpdateCondition(
			drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionOwned),
			metav1.ConditionFalse,
			string(directcsi.DirectCSIDriveReasonAdded),
			fmt.Sprintf("failed to release drive: %s", err.Error()),
		)
		_, uerr := client.GetLatestDirectCSIDriveInterface().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
		)
		if uerr != nil && err == nil {
			err = uerr
		}
	}
	return err
}

func (handler *driveEventHandler) mountDrive(ctx context.Context, drive *directcsi.DirectCSIDrive, remount bool) error {
	device, err := handler.getDevice(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		klog.Error(err)
		return err
	}
	if remount {
		err = handler.unmountDevice(device)
		if err != nil {
			klog.Errorf("failed to umount drive %s; %w", drive.Name, err)
		}
	}
	if err == nil {
		target := filepath.Join(sys.MountRoot, drive.Name)
		err = handler.mountDevice(device, target, drive.Spec.RequestedFormat.MountOptions)
		if err != nil {
			klog.Errorf("failed to mount drive %s; %w", drive.Name, err)
			utils.UpdateCondition(
				drive.Status.Conditions,
				string(directcsi.DirectCSIDriveConditionOwned),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIDriveReasonAdded),
				fmt.Sprintf("failed to mount drive: %s", err.Error()),
			)
			_, uerr := client.GetLatestDirectCSIDriveInterface().Update(
				ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
			)
			if uerr != nil && err == nil {
				err = uerr
			}
		}
	}
	return err
}

func (handler *driveEventHandler) lost(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	// Set the drive ready condition as false if not set
	if !utils.IsConditionStatus(drive.Status.Conditions, string(directcsi.DirectCSIDriveConditionReady), metav1.ConditionFalse) {
		utils.UpdateCondition(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionReady),
			metav1.ConditionFalse,
			string(directcsi.DirectCSIDriveReasonLost),
			"drive is removed")
		_, err := client.GetLatestDirectCSIDriveInterface().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
		)
		if err != nil {
			return err
		}
	}
	// update the volume conditions to be not ready
	if drive.Status.DriveStatus == directcsi.DriveStatusInUse {
		volumeInterface := client.GetLatestDirectCSIVolumeInterface()
		for _, finalizer := range drive.GetFinalizers() {
			if !strings.HasPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix) {
				continue
			}
			volumeName := strings.TrimPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix)
			volume, err := volumeInterface.Get(
				ctx, volumeName, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
			utils.UpdateCondition(volume.Status.Conditions,
				string(directcsi.DirectCSIVolumeConditionReady),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIVolumeReasonNotReady),
				"[DRIVE LOST]",
			)
			_, err = volumeInterface.Update(
				ctx, volume, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
