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

package discovery

import (
	"context"
	"path/filepath"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (d *Discovery) verifyDriveMount(existingDrive *directcsi.DirectCSIDrive) error {
	driveMounter := &sys.DefaultDriveMounter{}
	switch existingDrive.Status.DriveStatus {
	case directcsi.DriveStatusInUse, directcsi.DriveStatusReady:
		mountSource := sys.GetDirectCSIPath(existingDrive.Status.FilesystemUUID)
		mountTarget := filepath.Join(sys.MountRoot, existingDrive.Status.FilesystemUUID)
		// Check if the drive is mounted
		isMounted := false
		for _, mount := range d.mounts {
			if mount.MountSource == mountSource {
				isMounted = true
				break
			}
		}
		// Mount if umounted
		if !isMounted {
			if err := driveMounter.MountDrive(mountSource, mountTarget, []string{}); err != nil {
				return err
			}
			existingDrive.Status.Mountpoint = mountTarget
		}
	}
	return nil
}

func syncDriveStatesOnDiscovery(existingObj *directcsi.DirectCSIDrive, localDrive *directcsi.DirectCSIDrive) {

	existingObjVersion := utils.GetLabelV(existingObj, utils.VersionLabel)
	// overwrite existing object labels
	existingObj.SetLabels(localDrive.GetLabels())
	utils.UpdateLabels(existingObj,
		utils.AccessTierLabel, string(existingObj.Status.AccessTier), // set access-tier labels
		utils.VersionLabel, existingObjVersion, // set obj version labels
	)

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
	// Capacity sync
	allocatedCapacity := localDrive.Status.AllocatedCapacity
	if existingObj.Status.DriveStatus == directcsi.DriveStatusInUse {
		// size reserved for allocated volumes
		allocatedCapacity = existingObj.Status.AllocatedCapacity
	}
	existingObj.Status.FreeCapacity = localDrive.Status.TotalCapacity - allocatedCapacity
	existingObj.Status.AllocatedCapacity = allocatedCapacity
}

func (d *Discovery) syncDrive(ctx context.Context, localDrive *directcsi.DirectCSIDrive) error {
	directCSIClient := d.directcsiClient.DirectV1beta2()
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
		if err := d.verifyDriveMount(existingDrive); err != nil {
			utils.UpdateCondition(existingDrive.Status.Conditions,
				string(directcsi.DirectCSIDriveConditionInitialized),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIDriveReasonInitialized),
				err.Error())
			klog.V(3).Infof("mounting failed with: %v", err)
		}

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
	directCSIClient := d.directcsiClient.DirectV1beta2()
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
