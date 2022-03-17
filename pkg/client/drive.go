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

package client

import (
	"context"
	"fmt"
	"path"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func isDirectCSIMount(mountPoints []string) bool {
	if len(mountPoints) == 0 {
		return true
	}

	for _, mountPoint := range mountPoints {
		if strings.HasPrefix(mountPoint, "/var/lib/direct-csi/") {
			return true
		}
	}
	return false
}

// NewDirectCSIDriveStatus creates direct CSI drive status.
func NewDirectCSIDriveStatus(device *sys.Device, nodeID string, topology map[string]string) directcsi.DirectCSIDriveStatus {
	driveStatus := directcsi.DriveStatusAvailable
	if device.Size < sys.MinSupportedDeviceSize ||
		device.SwapOn ||
		device.Hidden ||
		device.ReadOnly ||
		device.Removable ||
		device.Partitioned ||
		device.Master != "" ||
		len(device.Holders) > 0 ||
		!isDirectCSIMount(device.MountPoints) {
		driveStatus = directcsi.DriveStatusUnavailable
	}

	mounted := metav1.ConditionFalse
	if device.FirstMountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	formatted := metav1.ConditionFalse
	if device.FSType != "" {
		formatted = metav1.ConditionTrue
	}

	return directcsi.DirectCSIDriveStatus{
		AccessTier:        directcsi.AccessTierUnknown,
		DriveStatus:       driveStatus,
		Filesystem:        device.FSType,
		FreeCapacity:      int64(device.FreeCapacity),
		AllocatedCapacity: int64(device.Size - device.FreeCapacity),
		LogicalBlockSize:  int64(device.LogicalBlockSize),
		ModelNumber:       device.Model,
		MountOptions:      device.FirstMountOptions,
		Mountpoint:        device.FirstMountPoint,
		NodeName:          nodeID,
		PartitionNum:      device.Partition,
		Path:              "/dev/" + device.Name,
		PhysicalBlockSize: int64(device.PhysicalBlockSize),
		RootPartition:     device.Name,
		SerialNumber:      device.Serial,
		TotalCapacity:     int64(device.Size),
		FilesystemUUID:    device.FSUUID,
		PartitionUUID:     device.PartUUID,
		MajorNumber:       uint32(device.Major),
		MinorNumber:       uint32(device.Minor),
		Topology:          topology,
		UeventSerial:      device.UeventSerial,
		UeventFSUUID:      device.UeventFSUUID,
		WWID:              device.WWID,
		Vendor:            device.Vendor,
		DMName:            device.DMName,
		DMUUID:            device.DMUUID,
		MDUUID:            device.MDUUID,
		PartTableUUID:     device.PTUUID,
		PartTableType:     device.PTType,
		Virtual:           device.Virtual,
		ReadOnly:          device.ReadOnly,
		Partitioned:       device.Partitioned,
		SwapOn:            device.SwapOn,
		Master:            device.Master,
		Conditions: []metav1.Condition{
			{
				Type:               string(directcsi.DirectCSIDriveConditionOwned),
				Status:             metav1.ConditionFalse,
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionMounted),
				Status:             mounted,
				Message:            device.FirstMountPoint,
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionFormatted),
				Status:             formatted,
				Message:            "xfs",
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionInitialized),
				Status:             metav1.ConditionTrue,
				Message:            "",
				Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionReady),
				Status:             metav1.ConditionTrue,
				Message:            "",
				Reason:             string(directcsi.DirectCSIDriveReasonReady),
				LastTransitionTime: metav1.Now(),
			},
		},
	}
}

// NewDirectCSIDrive creates new direct-csi drive.
func NewDirectCSIDrive(name string, status directcsi.DirectCSIDriveStatus) *directcsi.DirectCSIDrive {
	drive := &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     status,
	}

	utils.UpdateLabels(drive, map[utils.LabelKey]utils.LabelValue{
		utils.NodeLabelKey:       utils.NewLabelValue(status.NodeName),
		utils.PathLabelKey:       utils.NewLabelValue(utils.SanitizeDrivePath(status.Path)),
		utils.VersionLabelKey:    utils.NewLabelValue(directcsi.Version),
		utils.CreatedByLabelKey:  utils.DirectCSIDriverName,
		utils.AccessTierLabelKey: utils.NewLabelValue(string(status.AccessTier)),
	})

	return drive
}

// CreateDrive creates drive CRD.
func CreateDrive(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	_, err := latestDirectCSIDriveInterface.Create(ctx, drive, metav1.CreateOptions{})
	return err
}

// DeleteDrive deletes drive CRD.
func DeleteDrive(
	ctx context.Context,
	drive *directcsi.DirectCSIDrive,
	force bool) error {
	var err error
	if drive.Status.DriveStatus != directcsi.DriveStatusTerminating {
		drive.Status.DriveStatus = directcsi.DriveStatusTerminating
		drive, err = latestDirectCSIDriveInterface.Update(ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
		if err != nil {
			return err
		}
	}

	if force {
		if drive.Status.FilesystemUUID != "" && drive.Status.DriveStatus != directcsi.DriveStatusInUse {
			if err := mount.Unmount(path.Join(sys.MountRoot, drive.Status.FilesystemUUID), true, true, false); err != nil {
				klog.Errorf("unable to unmount %v; %v", path.Join(sys.MountRoot, drive.Status.FilesystemUUID), err)
			}
		}
		return latestDirectCSIDriveInterface.Delete(ctx, drive.Name, metav1.DeleteOptions{})
	}

	finalizers := drive.GetFinalizers()
	switch len(finalizers) {
	case 1:
		if finalizers[0] != directcsi.DirectCSIDriveFinalizerDataProtection {
			return fmt.Errorf("invalid state reached. Report this issue at https://github.com/minio/directpv/issues")
		}

		if err := mount.SafeUnmount(path.Join(sys.MountRoot, drive.Status.FilesystemUUID), false, false, false); err != nil {
			return err
		}

		drive.Finalizers = []string{}
		_, err := latestDirectCSIDriveInterface.Update(ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
		return err
	case 0:
		return nil
	default:
		for _, finalizer := range finalizers {
			if !strings.HasPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix) {
				continue
			}
			volumeName := strings.TrimPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix)
			volume, err := latestDirectCSIVolumeInterface.Get(
				ctx, volumeName, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
			utils.UpdateCondition(volume.Status.Conditions,
				string(directcsi.DirectCSIVolumeConditionReady),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIVolumeReasonNotReady),
				"[DRIVE LOST] Please refer https://github.com/minio/directpv/blob/master/docs/troubleshooting.md",
			)
			_, err = latestDirectCSIVolumeInterface.Update(
				ctx, volume, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
