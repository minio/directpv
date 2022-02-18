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
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
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

func newDriveEventHandler(nodeID string, reflinkSupport bool) *driveEventHandler {
	return &driveEventHandler{
		nodeID:          nodeID,
		reflinkSupport:  reflinkSupport,
		getDevice:       getDevice,
		stat:            os.Stat,
		mountDevice:     mount.MountXFSDevice,
		unmountDevice:   mount.UnmountDevice,
		makeFS:          xfs.MakeFS,
		getFreeCapacity: getFreeCapacity,
	}
}

func (handler *driveEventHandler) ListerWatcher() cache.ListerWatcher {
	return utils.DrivesListerWatcher(handler.nodeID)
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
	case listener.AddEvent, listener.UpdateEvent:
		return handler.update(ctx, args.Object.(*directcsi.DirectCSIDrive))
	case listener.DeleteEvent:
		return handler.delete(ctx, args.Object.(*directcsi.DirectCSIDrive))
	}
	return nil
}

func (handler *driveEventHandler) getFSUUID(ctx context.Context, drive *directcsi.DirectCSIDrive) (string, error) {
	if drive.Status.FilesystemUUID == "" ||
		drive.Status.FilesystemUUID == "00000000-0000-0000-0000-000000000000" ||
		len(drive.Status.FilesystemUUID) != 36 {
		return uuid.New().String(), nil
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// Use new UUID if it is aleady used in another drive.
	resultCh, err := client.ListDrives(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(handler.nodeID)},
		nil,
		nil,
		client.MaxThreadCount,
	)
	if err != nil {
		return "", err
	}

	for result := range resultCh {
		if result.Err != nil {
			return "", result.Err
		}

		if result.Drive.Name != drive.Name && result.Drive.Status.FilesystemUUID == drive.Status.FilesystemUUID {
			return uuid.New().String(), nil
		}
	}

	return drive.Status.FilesystemUUID, nil
}

func (handler *driveEventHandler) format(ctx context.Context, drive *directcsi.DirectCSIDrive) (err error) {
	fsUUID, err := handler.getFSUUID(ctx, drive)
	if err != nil {
		klog.Error(err)
		return err
	}
	drive.Status.FilesystemUUID = fsUUID

	if drive.Status.DriveStatus == directcsi.DriveStatusInUse {
		err = fmt.Errorf("formatting drive %v in InUse state is not allowed", drive.Name)
		klog.Error(err)
		return err
	}

	device, err := handler.getDevice(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		klog.Error(err)
		return err
	}
	if _, err := handler.stat(device); err != nil {
		err = fmt.Errorf("unable to read device %v of major/minor %v:%v; %w", device, drive.Status.MajorNumber, drive.Status.MinorNumber, err)
		klog.Error(err)
		return err
	}

	target := filepath.Join(sys.MountRoot, drive.Status.FilesystemUUID)
	mountOpts := drive.Spec.RequestedFormat.MountOptions
	force := drive.Spec.RequestedFormat.Force
	mounted := drive.Status.Mountpoint != ""
	formatted := drive.Status.Filesystem != ""

	if err == nil && (!formatted || force) {
		if mounted {
			if err = handler.unmountDevice(device); err != nil {
				klog.Errorf("failed to unmount drive %s; %w", drive.Name, err)
			} else {
				drive.Status.Mountpoint = ""
				mounted = false
			}
		}

		if err == nil {
			if err = handler.makeFS(ctx, drive.Status.Path, drive.Status.FilesystemUUID, force, handler.reflinkSupport); err != nil {
				klog.Errorf("failed to format drive %s; %w", drive.Name, err)
			} else {
				drive.Status.Filesystem = "xfs"
				drive.Status.AllocatedCapacity = 0
				formatted = true
			}
		}
	}

	if err == nil && formatted && !mounted {
		if err = handler.mountDevice(device, target, mountOpts); err != nil {
			klog.Error("failed to mount drive %s; %w", drive.Name, err)
		} else {
			drive.Status.Mountpoint = target
			drive.Status.MountOptions = mountOpts
			freeCapacity := uint64(0)
			if freeCapacity, err = handler.getFreeCapacity(drive.Status.Mountpoint); err != nil {
				klog.Errorf("unable to get free capacity of %v; %v", drive.Status.Mountpoint, err)
			} else {
				mounted = true
				drive.Status.FreeCapacity = int64(freeCapacity)
				drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity
			}
		}
	}

	message := ""
	if err != nil {
		message = err.Error()
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		utils.BoolToCondition(formatted && mounted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	message = string(directcsi.DirectCSIDriveMessageNotMounted)
	if mounted {
		message = string(directcsi.DirectCSIDriveMessageMounted)
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionMounted),
		utils.BoolToCondition(mounted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	message = string(directcsi.DirectCSIDriveMessageNotFormatted)
	if formatted {
		message = string(directcsi.DirectCSIDriveMessageFormatted)
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionFormatted),
		utils.BoolToCondition(formatted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	if err == nil {
		drive.Finalizers = []string{directcsi.DirectCSIDriveFinalizerDataProtection}
		drive.Status.DriveStatus = directcsi.DriveStatusReady
		drive.Spec.RequestedFormat = nil
	}

	_, uerr := client.GetLatestDirectCSIDriveInterface().Update(
		ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
	)
	if uerr != nil {
		if err == nil {
			err = uerr
		} else {
			klog.V(5).ErrorS(err, "unable to update drive", "name", drive.Name)
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
	if err = handler.unmountDevice(device); err != nil {
		err = fmt.Errorf("failed to release drive %s; %w", drive.Name, err)
		klog.Error(err)
	} else {
		drive.Status.DriveStatus = directcsi.DriveStatusAvailable
		drive.Finalizers = []string{}
		utils.UpdateCondition(
			drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionMounted),
			metav1.ConditionFalse,
			string(directcsi.DirectCSIDriveReasonAdded),
			"",
		)
	}

	message := ""
	if err != nil {
		message = err.Error()
	}

	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		metav1.ConditionFalse,
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	_, uerr := client.GetLatestDirectCSIDriveInterface().Update(
		ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
	)
	if uerr != nil {
		if err == nil {
			err = uerr
		} else {
			klog.V(5).ErrorS(err, "unable to update drive", "name", drive.Name)
		}
	}

	return err
}

func (handler *driveEventHandler) update(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	klog.V(5).Infof("drive update called on %s", drive.Name)

	// Release the drive
	if drive.Status.DriveStatus == directcsi.DriveStatusReleased {
		klog.V(3).Infof("releasing drive %s", drive.Name)
		return handler.release(ctx, drive)
	}

	// Format the drive
	if drive.Spec.DirectCSIOwned && drive.Spec.RequestedFormat != nil {
		klog.V(3).Infof("owning and formatting drive %s", drive.Name)

		switch drive.Status.DriveStatus {
		case directcsi.DriveStatusAvailable:
			return handler.format(ctx, drive)
		case directcsi.DriveStatusReleased,
			directcsi.DriveStatusUnavailable,
			directcsi.DriveStatusReady,
			directcsi.DriveStatusTerminating,
			directcsi.DriveStatusInUse:
			klog.V(3).Infof("rejecting to format drive %s due to %s", drive.Name, drive.Status.DriveStatus)
		}
	}

	return nil
}

func (handler *driveEventHandler) delete(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	return client.DeleteDrive(ctx, drive, false)
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

func getDevice(major, minor uint32) (string, error) {
	name, err := sys.GetDeviceName(major, minor)
	if err != nil {
		return "", err
	}
	return "/dev/" + name, nil
}
