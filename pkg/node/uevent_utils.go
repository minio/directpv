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

package node

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

var (
	errInvalidMount = errors.New("the directpv mount is not found")
	errInvalidDrive = func(fieldName string, expected, found interface{}) error {
		return fmt.Errorf("; %s mismatch - Expected %v found %v",
			fieldName,
			expected,
			found)
	}
)

func (d *driveEventHandler) updateDrive(device *sys.Device, drive *directcsi.DirectCSIDrive) (*directcsi.DirectCSIDrive, error) {
	err := validateDrive(drive, device)
	return d.setDriveStatus(device, drive), err
}

func (d *driveEventHandler) setDriveStatus(device *sys.Device, drive *directcsi.DirectCSIDrive) *directcsi.DirectCSIDrive {
	updatedDrive := drive.DeepCopy()
	updatedDrive.Status.NodeName = d.nodeID
	updatedDrive.Status.Topology = d.topology
	updatedDrive.Status.Filesystem = device.FSType
	updatedDrive.Status.FilesystemUUID = device.FSUUID
	updatedDrive.Status.UeventFSUUID = device.UeventFSUUID
	updatedDrive.Status.MajorNumber = uint32(device.Major)
	updatedDrive.Status.MinorNumber = uint32(device.Minor)
	updatedDrive.Status.Path = device.DevPath()
	updatedDrive.Status.LogicalBlockSize = int64(device.LogicalBlockSize)
	updatedDrive.Status.MountOptions = device.FirstMountOptions
	updatedDrive.Status.Mountpoint = device.FirstMountPoint
	updatedDrive.Status.DMName = device.DMName
	updatedDrive.Status.ReadOnly = device.ReadOnly
	updatedDrive.Status.RootPartition = device.Name
	updatedDrive.Status.Virtual = device.Virtual
	updatedDrive.Status.SwapOn = device.SwapOn
	updatedDrive.Status.Master = device.Master
	updatedDrive.Status.PartTableUUID = device.PTUUID
	updatedDrive.Status.PartTableType = device.PTType
	updatedDrive.Status.Partitioned = device.Partitioned

	// fill hwinfo only if it is empty
	if updatedDrive.Status.PartitionUUID == "" {
		updatedDrive.Status.PartitionUUID = device.PartUUID
	}
	if updatedDrive.Status.DMUUID == "" {
		updatedDrive.Status.DMUUID = device.DMUUID
	}
	if updatedDrive.Status.MDUUID == "" {
		updatedDrive.Status.MDUUID = device.MDUUID
	}
	if updatedDrive.Status.PartitionNum == int(0) {
		updatedDrive.Status.PartitionNum = device.Partition
	}
	if updatedDrive.Status.PhysicalBlockSize == int64(0) {
		updatedDrive.Status.PhysicalBlockSize = int64(device.PhysicalBlockSize)
	}
	if updatedDrive.Status.ModelNumber == "" {
		updatedDrive.Status.ModelNumber = device.Model
	}
	if updatedDrive.Status.SerialNumber == "" {
		updatedDrive.Status.SerialNumber = device.Serial
	}
	if updatedDrive.Status.UeventSerial == "" {
		updatedDrive.Status.UeventSerial = device.UeventSerial
	}
	if updatedDrive.Status.WWID == "" {
		updatedDrive.Status.WWID = device.WWID
	}
	if updatedDrive.Status.Vendor == "" {
		updatedDrive.Status.Vendor = device.Vendor
	}

	if device.ReadOnly || device.Partitioned || device.SwapOn || device.Master != "" || !validDirectPVMounts(device.MountPoints) {
		if updatedDrive.Status.DriveStatus == directcsi.DriveStatusAvailable {
			updatedDrive.Status.DriveStatus = directcsi.DriveStatusUnavailable
		}
	} else {
		if updatedDrive.Status.DriveStatus == directcsi.DriveStatusUnavailable {
			updatedDrive.Status.DriveStatus = directcsi.DriveStatusAvailable
		}
	}

	// update the path and respective label value
	updatedDrive.Status.Path = device.DevPath()
	utils.UpdateLabels(updatedDrive, map[utils.LabelKey]utils.LabelValue{
		utils.PathLabelKey: utils.NewLabelValue(utils.SanitizeDrivePath(device.Name)),
	})

	// capacity sync
	updatedDrive.Status.TotalCapacity = int64(device.Size)
	if updatedDrive.Status.DriveStatus != directcsi.DriveStatusInUse {
		updatedDrive.Status.AllocatedCapacity = int64(device.Size - device.FreeCapacity)
	}
	updatedDrive.Status.FreeCapacity = updatedDrive.Status.TotalCapacity - updatedDrive.Status.AllocatedCapacity

	return updatedDrive
}

func validateDrive(drive *directcsi.DirectCSIDrive, device *sys.Device) error {
	var err error
	switch drive.Status.DriveStatus {
	case directcsi.DriveStatusInUse, directcsi.DriveStatusReady:
		if !validDirectPVMounts(device.MountPoints) {
			err = multierr.Append(err, errInvalidMount)
		}
		if device.FirstMountPoint != filepath.Join(sys.MountRoot, drive.Name) {
			err = multierr.Append(err, errInvalidDrive(
				"Mountpoint",
				filepath.Join(sys.MountRoot, drive.Name),
				device.FirstMountPoint))
		}
		if !validDirectPVMountOpts(device.FirstMountOptions) {
			err = multierr.Append(err, errInvalidDrive(
				"MountpointOptions",
				mount.MountOptPrjQuota,
				device.FirstMountOptions))
		}
		if drive.Status.UeventFSUUID != device.UeventFSUUID {
			err = multierr.Append(err, errInvalidDrive(
				"UeventFSUUID",
				drive.Status.UeventFSUUID,
				device.UeventFSUUID))
		}
		if drive.Status.FilesystemUUID != device.FSUUID {
			err = multierr.Append(err, errInvalidDrive(
				"FilesystemUUID",
				drive.Status.FilesystemUUID,
				device.FSUUID))
		}
		if drive.Status.Filesystem != device.FSType {
			err = multierr.Append(err, errInvalidDrive(
				"Filesystem",
				drive.Status.Filesystem,
				device.FSType))
		}
		if device.Size < sys.MinSupportedDeviceSize {
			err = multierr.Append(err, fmt.Errorf(
				"the size of the drive is less than %v",
				sys.MinSupportedDeviceSize))
		}
		if device.ReadOnly {
			err = multierr.Append(err, errInvalidDrive(
				"ReadOnly",
				false,
				device.ReadOnly))
		}
		if device.SwapOn {
			err = multierr.Append(err, errInvalidDrive(
				"SwapOn",
				false,
				device.SwapOn))
		}
		if device.Master != "" {
			err = multierr.Append(err, errInvalidDrive(
				"Master",
				"",
				device.Master))
		}
	}
	return err
}

func syncVolumeLabels(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	volumeInterface := client.GetLatestDirectCSIVolumeInterface()
	updateLabels := func(volumeName, driveName string) func() error {
		return func() error {
			volume, err := volumeInterface.Get(
				ctx, volumeName, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
			volume.Labels[string(utils.DrivePathLabelKey)] = driveName
			_, err = volumeInterface.Update(
				ctx, volume, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
			)
			return err
		}
	}
	for _, finalizer := range drive.GetFinalizers() {
		if !strings.HasPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix) {
			continue
		}
		volumeName := strings.TrimPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix)
		if err := retry.RetryOnConflict(
			retry.DefaultRetry,
			updateLabels(volumeName, utils.SanitizeDrivePath(drive.Status.Path))); err != nil {
			klog.ErrorS(err, "unable to update volume %v", volumeName)
			return err
		}
	}
	return nil
}

func validDirectPVMounts(mountPoints []string) bool {
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

func validDirectPVMountOpts(deviceMountOpts []string) bool {
	expectedMountOpts := []string{
		mount.MountOptPrjQuota,
	}
	for _, expectedMountOpt := range expectedMountOpts {
		foundExpectedOpt := false
		for _, deviceMountOpt := range deviceMountOpts {
			if deviceMountOpt == expectedMountOpt {
				foundExpectedOpt = true
				break
			}
		}
		if !foundExpectedOpt {
			return false
		}
	}
	return true
}
