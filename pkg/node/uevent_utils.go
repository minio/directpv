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
	"path"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
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
	updatedDrive := d.setDriveStatus(device, drive)
	return updatedDrive, validateDrive(updatedDrive, device)
}

func (d *driveEventHandler) setDriveStatus(device *sys.Device, drive *directcsi.DirectCSIDrive) *directcsi.DirectCSIDrive {
	updatedDrive := drive.DeepCopy()
	updatedDrive.Status.NodeName = d.nodeID
	updatedDrive.Status.Topology = d.topology
	updatedDrive.Status.UeventFSUUID = device.UeventFSUUID
	updatedDrive.Status.MajorNumber = uint32(device.Major)
	updatedDrive.Status.MinorNumber = uint32(device.Minor)
	updatedDrive.Status.Path = device.DevPath()
	updatedDrive.Status.LogicalBlockSize = int64(device.LogicalBlockSize)

	updatedDrive.Status.DMName = device.DMName
	updatedDrive.Status.ReadOnly = device.ReadOnly
	updatedDrive.Status.RootPartition = device.Name
	updatedDrive.Status.Virtual = device.Virtual
	updatedDrive.Status.SwapOn = device.SwapOn
	updatedDrive.Status.Master = device.Master
	updatedDrive.Status.PartTableUUID = device.PTUUID
	updatedDrive.Status.PartTableType = device.PTType
	updatedDrive.Status.Partitioned = device.Partitioned
	updatedDrive.Status.PCIPath = device.PCIPath

	// do not update FS info for managed drives
	if updatedDrive.Status.Filesystem == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.Filesystem = device.FSType
	}
	if updatedDrive.Status.FilesystemUUID == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.FilesystemUUID = device.FSUUID
	}

	// populate mount infos
	updatedDrive.Status.MountOptions = device.FirstMountOptions
	updatedDrive.Status.Mountpoint = device.FirstMountPoint
	// other mounts
	var otherMountsInfo []directcsi.OtherMountsInfo
	for _, mountInfo := range device.OtherMountsInfo {
		otherMountsInfo = append(otherMountsInfo, directcsi.OtherMountsInfo{
			Mountpoint:   mountInfo.MountPoint,
			MountOptions: mountInfo.MountOptions,
		})
	}
	updatedDrive.Status.OtherMountsInfo = otherMountsInfo

	// fill hwinfo only if it is empty
	if updatedDrive.Status.PartitionUUID == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.PartitionUUID = device.PartUUID
	} else {
		// PartitionUUID values in versions < 3.0.0 is upper-cased
		if strings.EqualFold(updatedDrive.Status.PartitionUUID, device.PartUUID) {
			updatedDrive.Status.PartitionUUID = device.PartUUID
		}
		// bugfix: for versions < 3.0.0, the partitionUUID has to be unset or set to empty for root partitions
		if device.Partition == int(0) {
			updatedDrive.Status.PartitionUUID = device.PartUUID
		}
	}

	if updatedDrive.Status.DMUUID == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.DMUUID = device.DMUUID
	}
	if updatedDrive.Status.MDUUID == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.MDUUID = device.MDUUID
	}
	if updatedDrive.Status.PartitionNum == int(0) || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.PartitionNum = device.Partition
	}
	if updatedDrive.Status.PhysicalBlockSize == int64(0) || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.PhysicalBlockSize = int64(device.PhysicalBlockSize)
	}
	if updatedDrive.Status.ModelNumber == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.ModelNumber = device.Model
	}
	if updatedDrive.Status.SerialNumber == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.SerialNumber = device.Serial
	}
	if updatedDrive.Status.SerialNumberLong == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.SerialNumberLong = device.SerialLong
	}
	if updatedDrive.Status.UeventSerial == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.UeventSerial = device.UeventSerial
	}
	if updatedDrive.Status.WWID == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.WWID = device.WWID
	}
	if updatedDrive.Status.Vendor == "" || !utils.IsManagedDrive(updatedDrive) {
		updatedDrive.Status.Vendor = device.Vendor
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

	// check and update if the requests succeeded
	checkAndUpdateConditions(updatedDrive, device)

	return updatedDrive
}

func validateDrive(drive *directcsi.DirectCSIDrive, device *sys.Device) error {
	var err error
	switch drive.Status.DriveStatus {
	case directcsi.DriveStatusInUse, directcsi.DriveStatusReady:
		// Check if the drive is umounted or if the directpv mount is not found
		if device.FirstMountPoint == "" || !mount.ValidDirectPVMounts(device.MountPoints) {
			err = multierr.Append(err, errInvalidMount)
		}
		// Verify drive mount and mountopts
		if device.FirstMountPoint != "" {
			if device.FirstMountPoint != path.Join(sys.MountRoot, drive.Name) &&
				device.FirstMountPoint != path.Join(sys.MountRoot, drive.Status.FilesystemUUID) {
				err = multierr.Append(err, errInvalidDrive(
					"Mountpoint",
					path.Join(sys.MountRoot, drive.Name),
					device.FirstMountPoint))
			}
			if !mount.ValidDirectPVMountOpts(device.FirstMountOptions) {
				err = multierr.Append(err, errInvalidDrive(
					"MountpointOptions",
					mount.MountOptRW,
					device.FirstMountOptions))
			}
		}
		// Check other device attributes
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
		if device.Hidden {
			err = multierr.Append(err, errInvalidDrive(
				"Hidden",
				false,
				device.Hidden))
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
		if len(device.Holders) > 0 {
			err = multierr.Append(err, fmt.Errorf(
				"the device has holders: %v",
				device.Holders))
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
			if volume.Labels != nil {
				if val, ok := volume.Labels[string(utils.DrivePathLabelKey)]; ok && val == driveName {
					// no change to the label value
					return nil
				}
			} else {
				volume.Labels = map[string]string{}
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

func checkAndUpdateConditions(drive *directcsi.DirectCSIDrive, device *sys.Device) {
	switch drive.Status.DriveStatus {
	case directcsi.DriveStatusAvailable:
		// Check if formatting request succeeded, If so, update the status fields
		if drive.Spec.RequestedFormat != nil {
			if drive.Status.Mountpoint == path.Join(sys.MountRoot, drive.Name) {
				drive.Finalizers = []string{directcsi.DirectCSIDriveFinalizerDataProtection}
				drive.Status.DriveStatus = directcsi.DriveStatusReady
				drive.Spec.RequestedFormat = nil
				utils.UpdateCondition(
					drive.Status.Conditions,
					string(directcsi.DirectCSIDriveConditionOwned),
					metav1.ConditionTrue,
					string(directcsi.DirectCSIDriveReasonAdded),
					"",
				)
				break
			}
		}
		if sys.IsDeviceUnavailable(device) {
			drive.Status.DriveStatus = directcsi.DriveStatusUnavailable
		}
	// Check if release request succeeded
	case directcsi.DriveStatusReleased:
		if drive.Status.Mountpoint == "" {
			drive.Status.DriveStatus = directcsi.DriveStatusAvailable
			drive.Finalizers = []string{}
			utils.UpdateCondition(
				drive.Status.Conditions,
				string(directcsi.DirectCSIDriveConditionOwned),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIDriveReasonAdded),
				"",
			)
		}
	case directcsi.DriveStatusUnavailable:
		if !sys.IsDeviceUnavailable(device) {
			drive.Status.DriveStatus = directcsi.DriveStatusAvailable
		}
	}

	mountCondition := utils.BoolToCondition(drive.Status.Mountpoint != "")
	formattedCondition := utils.BoolToCondition(drive.Status.Filesystem != "")
	if !utils.IsConditionStatus(drive.Status.Conditions, string(directcsi.DirectCSIDriveConditionMounted), mountCondition) {
		utils.UpdateCondition(
			drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionMounted),
			mountCondition,
			string(directcsi.DirectCSIDriveReasonAdded),
			"",
		)
	}
	if !utils.IsConditionStatus(drive.Status.Conditions, string(directcsi.DirectCSIDriveConditionFormatted), formattedCondition) {
		utils.UpdateCondition(
			drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionFormatted),
			formattedCondition,
			string(directcsi.DirectCSIDriveReasonAdded),
			"",
		)
	}
}
