// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package utils

import (
	"context"
	"fmt"
	"path"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if device.Size < 1048576 || device.ReadOnly || device.Partitioned || device.SwapOn || device.Master != "" || !isDirectCSIMount(device.MountPoints) {
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
		},
	}
}

// NewDirectCSIDrive creates new direct-csi drive.
func NewDirectCSIDrive(name string, status directcsi.DirectCSIDriveStatus) *directcsi.DirectCSIDrive {
	return &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				NodeLabel:       SanitizeLabelV(status.NodeName),
				DrivePathLabel:  SanitizeDrivePath(status.Path),
				VersionLabel:    directcsi.Version,
				CreatedByLabel:  "directcsi-driver",
				AccessTierLabel: string(status.AccessTier),
			},
		},
		Status: status,
	}
}

// CreateDrive creates drive CRD.
func CreateDrive(ctx context.Context, driveInterface clientset.DirectCSIDriveInterface, drive *directcsi.DirectCSIDrive) error {
	_, err := driveInterface.Create(ctx, drive, metav1.CreateOptions{})
	return err
}

// DeleteDrive deletes drive CRD.
func DeleteDrive(ctx context.Context, driveInterface clientset.DirectCSIDriveInterface, drive *directcsi.DirectCSIDrive, force bool) error {
	if drive.Status.DriveStatus != directcsi.DriveStatusTerminating {
		drive.Status.DriveStatus = directcsi.DriveStatusTerminating
		if _, err := driveInterface.Update(ctx, drive, metav1.UpdateOptions{TypeMeta: DirectCSIDriveTypeMeta()}); err != nil {
			return err
		}
	}

	if force {
		if drive.Status.FilesystemUUID != "" && drive.Status.DriveStatus != directcsi.DriveStatusInUse {
			sys.ForceUnmount(path.Join(sys.MountRoot, drive.Status.FilesystemUUID))
		}
		return driveInterface.Delete(ctx, drive.Name, metav1.DeleteOptions{})
	}

	switch finalizers := drive.GetFinalizers(); len(finalizers) {
	case 1:
		if finalizers[0] != directcsi.DirectCSIDriveFinalizerDataProtection {
			return fmt.Errorf("invalid state reached. Report this issue at https://github.com/minio/direct-csi/issues")
		}

		if err := sys.SafeUnmount(path.Join(sys.MountRoot, drive.Status.FilesystemUUID), nil); err != nil {
			return err
		}

		drive.Finalizers = []string{}
		_, err := driveInterface.Update(ctx, drive, metav1.UpdateOptions{TypeMeta: DirectCSIDriveTypeMeta()})
		return err
	case 0:
		return nil
	default:
		return fmt.Errorf("cannot delete drive in use")
	}
}
