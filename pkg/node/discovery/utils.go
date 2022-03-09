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

package discovery

import (
	"context"
	"fmt"
	"path/filepath"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *Discovery) getMountInfo(major, minor uint32) []mount.MountInfo {
	return d.mounts[fmt.Sprintf("%v:%v", major, minor)]
}

func (d *Discovery) verifyDriveMount(existingDrive *directcsi.DirectCSIDrive) error {
	switch existingDrive.Status.DriveStatus {
	case directcsi.DriveStatusInUse, directcsi.DriveStatusReady:
		mountTarget := filepath.Join(sys.MountRoot, existingDrive.Status.FilesystemUUID)

		mounted := false
		for _, mount := range d.getMountInfo(existingDrive.Status.MajorNumber, existingDrive.Status.MinorNumber) {
			if mounted = (mount.MountPoint == mountTarget); mounted {
				break
			}
		}

		// Mount if umounted
		if !mounted {
			name, err := sys.GetDeviceName(existingDrive.Status.MajorNumber, existingDrive.Status.MinorNumber)
			if err != nil {
				return err
			}
			if err = mount.Mount("/dev/"+name, mountTarget, "xfs", nil, mount.MountOptPrjQuota); err != nil {
				return err
			}
			existingDrive.Status.Mountpoint = mountTarget
		}
	}
	return nil
}

func syncDriveStatesOnDiscovery(existingObj *directcsi.DirectCSIDrive, localDrive *directcsi.DirectCSIDrive) {
	var existingVersion string
	if labels := existingObj.GetLabels(); labels != nil {
		existingVersion = labels[string(utils.VersionLabelKey)]
	}

	// overwrite existing object labels
	existingObj.SetLabels(localDrive.GetLabels())
	utils.UpdateLabels(existingObj, map[utils.LabelKey]utils.LabelValue{
		utils.AccessTierLabelKey: utils.NewLabelValue(string(existingObj.Status.AccessTier)),
		utils.VersionLabelKey:    utils.LabelValue(existingVersion),
	})

	// Sync the possible states
	existingObj.Status.RootPartition = localDrive.Status.RootPartition
	existingObj.Status.PartitionNum = localDrive.Status.PartitionNum
	existingObj.Status.Filesystem = localDrive.Status.Filesystem
	existingObj.Status.Mountpoint = localDrive.Status.Mountpoint
	existingObj.Status.MountOptions = localDrive.Status.MountOptions
	existingObj.Status.ModelNumber = localDrive.Status.ModelNumber
	existingObj.Status.PhysicalBlockSize = localDrive.Status.PhysicalBlockSize
	existingObj.Status.LogicalBlockSize = localDrive.Status.LogicalBlockSize
	existingObj.Status.Path = localDrive.Status.Path
	existingObj.Status.FilesystemUUID = localDrive.Status.FilesystemUUID
	existingObj.Status.SerialNumber = localDrive.Status.SerialNumber
	existingObj.Status.PartitionUUID = localDrive.Status.PartitionUUID
	existingObj.Status.MajorNumber = localDrive.Status.MajorNumber
	existingObj.Status.MinorNumber = localDrive.Status.MinorNumber
	existingObj.Status.TotalCapacity = localDrive.Status.TotalCapacity
	// drive status sync
	if localDrive.Status.DriveStatus == directcsi.DriveStatusUnavailable {
		existingObj.Status.DriveStatus = directcsi.DriveStatusUnavailable
		existingObj.Status.Conditions = localDrive.Status.Conditions
		existingObj.Spec.DirectCSIOwned = false
		existingObj.Spec.RequestedFormat = nil
	}
	if existingObj.Status.DriveStatus == directcsi.DriveStatusUnavailable && localDrive.Status.DriveStatus == directcsi.DriveStatusAvailable {
		existingObj.Status.DriveStatus = localDrive.Status.DriveStatus
	}
	// Capacity sync
	allocatedCapacity := localDrive.Status.AllocatedCapacity
	if existingObj.Status.DriveStatus == directcsi.DriveStatusInUse {
		// size reserved for allocated volumes
		allocatedCapacity = existingObj.Status.AllocatedCapacity
	}
	existingObj.Status.FreeCapacity = localDrive.Status.TotalCapacity - allocatedCapacity
	existingObj.Status.AllocatedCapacity = allocatedCapacity
	existingObj.Status.UeventSerial = localDrive.Status.UeventSerial
	existingObj.Status.UeventFSUUID = localDrive.Status.UeventFSUUID
	existingObj.Status.WWID = localDrive.Status.WWID
	existingObj.Status.Vendor = localDrive.Status.Vendor
	existingObj.Status.DMName = localDrive.Status.DMName
	existingObj.Status.DMUUID = localDrive.Status.DMUUID
	existingObj.Status.MDUUID = localDrive.Status.MDUUID
	existingObj.Status.PartTableUUID = localDrive.Status.PartTableUUID
	existingObj.Status.PartTableType = localDrive.Status.PartTableType
	existingObj.Status.ReadOnly = localDrive.Status.ReadOnly
	existingObj.Status.Partitioned = localDrive.Status.Partitioned
	existingObj.Status.SwapOn = localDrive.Status.SwapOn
	existingObj.Status.Master = localDrive.Status.Master
}

func (d *Discovery) syncDrive(ctx context.Context, localDrive *directcsi.DirectCSIDrive) error {
	directCSIClient := d.directcsiClient.DirectV1beta3()
	driveClient := directCSIClient.DirectCSIDrives()

	driveSync := func() error {
		existingDrive, err := driveClient.Get(ctx, localDrive.ObjectMeta.Name, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err != nil {
			return err
		}

		// Sync remote drive states
		syncDriveStatesOnDiscovery(existingDrive, localDrive)

		// Verify mounts
		message := ""
		if err := d.verifyDriveMount(existingDrive); err != nil {
			message = err.Error()
			klog.V(3).Infof("mounting failed with: %v", err)
		}
		utils.UpdateCondition(existingDrive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionInitialized),
			utils.BoolToCondition(message == ""),
			string(directcsi.DirectCSIDriveReasonInitialized),
			message)

		updateOpts := metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		}
		_, err = driveClient.Update(ctx, existingDrive, updateOpts)
		return err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, driveSync); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (d *Discovery) deleteUnmatchedRemoteDrives(ctx context.Context) error {
	directCSIClient := d.directcsiClient.DirectV1beta3()
	driveClient := directCSIClient.DirectCSIDrives()

	for _, remoteDrive := range d.remoteDrives {
		if remoteDrive.matched {
			continue
		}
		if err := driveClient.Delete(ctx, remoteDrive.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}
