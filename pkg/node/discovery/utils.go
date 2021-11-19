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
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"k8s.io/klog/v2"
)

func (d *Discovery) verifyDriveMount(existingDrive *directcsi.DirectCSIDrive) error {
	driveMounter := &sys.DefaultDriveMounter{}
	switch existingDrive.Status.DriveStatus {
	case directcsi.DriveStatusInUse, directcsi.DriveStatusReady:
		directCSIDevice := sys.GetDirectCSIPath(existingDrive.Status.FilesystemUUID)
		// verify and fix the (major, minor) if it has changed
		if err := sys.MakeBlockFile(directCSIDevice,
			existingDrive.Status.MajorNumber,
			existingDrive.Status.MinorNumber); err != nil {
			return err
		}

		mountTarget := filepath.Join(sys.MountRoot, existingDrive.Status.FilesystemUUID)
		// Check if the drive is mounted
		isMounted := false
		for _, mount := range d.mounts {
			if mount.MountSource == directCSIDevice {
				isMounted = true
				break
			}
		}
		// Mount if umounted
		if !isMounted {
			if err := driveMounter.MountDrive(directCSIDevice, mountTarget, []string{}); err != nil {
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
	if localDrive.Status.DriveStatus == directcsi.DriveStatusUnavailable {
		existingObj.Status.DriveStatus = directcsi.DriveStatusUnavailable
		existingObj.Status.Conditions = localDrive.Status.Conditions
		existingObj.Spec.DirectCSIOwned = false
		existingObj.Spec.RequestedFormat = nil
	}
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
		message := ""
		if err := d.verifyDriveMount(existingDrive); err != nil {
			message = err.Error()
			klog.V(3).Infof("mounting failed with: %v", err)
		}
		utils.UpdateCondition(existingDrive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionInitialized),
			utils.BoolToCondition(message == ""),
			string(directcsi.DirectCSIDriveReasonInitialized),
			message,
		)

		updateOpts := metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		}
		_, err = driveClient.Update(ctx, existingDrive, updateOpts)
		return err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, driveSync); err != nil {
		if !kerrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (d *Discovery) deleteUnmatchedRemoteDrives(ctx context.Context) error {
	// For unmatched drives, check the OS if the underlying drive is still active
	// of if it is lost.
	// If lost, then delete the unmapped remote drives
	// else, leave it running
	directCSIClient := d.directcsiClient.DirectV1beta2()
	driveClient := directCSIClient.DirectCSIDrives()

	for _, remoteDrive := range d.remoteDrives {
		if remoteDrive.matched {
			continue
		}
		if ok, err := d.verifyDriveLost(remoteDrive.DirectCSIDrive); !ok {
			klog.Errorf("drive %s still active, will not delete %v", remoteDrive.Name, err)
			continue
		}

		if err := driveClient.Delete(ctx, remoteDrive.Name, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (d *Discovery) verifyDriveLost(drive directcsi.DirectCSIDrive) (bool, error) {
	path := sys.GetDirectCSIPath(drive.Status.FilesystemUUID)

	stat, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	if stat.Mode() != fs.ModeDevice {
		return true, nil
	}

	mountPoint := drive.Status.Mountpoint

	mounts, errM := sys.ProbeMountInfo()
	if errM != nil {
		return false, errM
	}

	found := false
	for _, m := range mounts {
		if m.Mountpoint == mountPoint {
			found = true
			break
		}
	}

	if !found {
		return true, nil
	}

	fp := filepath.Join(mountPoint, ".drive_verifier")
	f, err := os.OpenFile(fp,
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return true, nil
	}
	defer func() {
		f.Close()
		os.RemoveAll(fp)
	}()

	if _, err := f.Write([]byte{0x00}); err != nil {
		return true, nil
	}
	return false, nil
}
