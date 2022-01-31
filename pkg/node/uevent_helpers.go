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
	"strings"

	directcsiv1beta1 "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta1"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func isDOSPTType(ptType string) bool {
	switch ptType {
	case "dos", "msdos", "mbr":
		return true
	default:
		return false
	}
}

func ptTypeEqual(ptType1, ptType2 string) bool {
	ptType1, ptType2 = strings.ToLower(ptType1), strings.ToLower(ptType2)
	switch {
	case ptType1 == ptType2:
		return true
	case isDOSPTType(ptType1) && isDOSPTType(ptType2):
		return true
	default:
		return false
	}
}

func isHWInfoAvailable(drive *directcsi.DirectCSIDrive) bool {
	return drive.Status.WWID != "" || drive.Status.SerialNumber != "" || drive.Status.UeventSerial != ""
}

func matchDeviceHWInfo(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.WWID != "" && drive.Status.WWID != device.WWID:
		return false
	case drive.Status.SerialNumber != "" && drive.Status.SerialNumber != device.Serial:
		return false
	case drive.Status.UeventSerial != "" && drive.Status.UeventSerial != device.UeventSerial:
		return false
	case drive.Status.ModelNumber != "" && drive.Status.ModelNumber != device.Model:
		return false
	case drive.Status.Vendor != "" && drive.Status.Vendor != device.Vendor:
		return false
	}

	return true
}

func isDMMDUUIDAvailable(drive *directcsi.DirectCSIDrive) bool {
	return drive.Status.DMUUID != "" || drive.Status.MDUUID != ""
}

func matchDeviceDMMDUUID(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.DMUUID != "" && drive.Status.DMUUID != device.DMUUID:
		return false
	case drive.Status.MDUUID != "" && drive.Status.MDUUID != device.MDUUID:
		return false
	}

	return true
}

func isPTUUIDAvailable(drive *directcsi.DirectCSIDrive) bool {
	return drive.Status.PartitionNum <= 0 && drive.Status.PartTableUUID != ""
}

func matchDevicePTUUID(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.PartTableUUID != device.PTUUID:
		return false
	case !ptTypeEqual(drive.Status.PartTableType, device.PTType):
		return false
	}

	return true
}

func isPartUUIDAvailable(drive *directcsi.DirectCSIDrive) bool {
	return drive.Status.PartitionNum > 0 && drive.Status.PartitionUUID != ""
}

func matchDevicePartUUID(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.PartitionUUID != device.PartUUID:
		return false
	}

	return true
}

func isFSUUIDAvailable(drive *directcsi.DirectCSIDrive) bool {
	return drive.Status.FilesystemUUID != "" || drive.Status.UeventFSUUID != ""
}

func matchDeviceFSUUID(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.FilesystemUUID != device.FSUUID:
		return false
	case drive.Status.UeventFSUUID != device.UeventFSUUID:
		return false
	case !sys.FSTypeEqual(drive.Status.Filesystem, device.FSType):
		return false
	}

	return true
}

func matchDeviceNameSize(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	switch {
	case drive.Status.PartitionNum != device.Partition:
		return false
	case drive.Status.Virtual != device.Virtual:
		return false
	case drive.Status.ReadOnly != device.ReadOnly:
		return false
	case uint64(drive.Status.TotalCapacity) != device.Size:
		return false
	case drive.Status.Path != "/dev/"+device.Name:
		return false
	}

	return true
}

func isV1Beta1Drive(drive *directcsi.DirectCSIDrive) bool {
	if labels := drive.GetLabels(); labels != nil {
		return labels[string(utils.VersionLabelKey)] == directcsiv1beta1.Version
	}
	return false
}

func matchV1Beta1Name(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	return drive.Status.Path == "/dev/"+device.Name
}

func updateDriveProperties(drive *directcsi.DirectCSIDrive, device *sys.Device) (bool, bool) {
	nameChanged := false
	updated := false

	if !sys.FSTypeEqual(drive.Status.Filesystem, device.FSType) {
		drive.Status.Filesystem = device.FSType
		updated = true
	}

	if drive.Status.TotalCapacity != int64(device.Size) {
		drive.Status.TotalCapacity = int64(device.Size)
		if drive.Status.AllocatedCapacity > drive.Status.TotalCapacity {
			drive.Status.AllocatedCapacity = drive.Status.TotalCapacity
		}
		drive.Status.FreeCapacity = drive.Status.TotalCapacity - drive.Status.AllocatedCapacity
		updated = true
	}

	if drive.Status.LogicalBlockSize != int64(device.LogicalBlockSize) {
		drive.Status.LogicalBlockSize = int64(device.LogicalBlockSize)
		updated = true
	}

	if drive.Status.ModelNumber != device.Model {
		drive.Status.ModelNumber = device.Model
		updated = true
	}

	if drive.Status.Mountpoint != device.FirstMountPoint {
		drive.Status.Mountpoint = device.FirstMountPoint
		drive.Status.MountOptions = device.FirstMountOptions
		updated = true
	}

	if drive.Status.PartitionNum != device.Partition {
		drive.Status.PartitionNum = device.Partition
		updated = true
	}

	if drive.Status.Path != "/dev/"+device.Name {
		drive.Status.Path = "/dev/" + device.Name
		if drive.Labels == nil {
			drive.Labels = map[string]string{}
		}
		drive.Labels[string(utils.DriveLabelKey)] = utils.SanitizeDrivePath(device.Name)
		nameChanged = true
		updated = true
	}

	if drive.Status.PhysicalBlockSize != int64(device.PhysicalBlockSize) {
		drive.Status.PhysicalBlockSize = int64(device.PhysicalBlockSize)
		updated = true
	}

	if drive.Status.RootPartition != device.Name {
		drive.Status.RootPartition = device.Name
		updated = true
	}

	if drive.Status.SerialNumber != device.Serial {
		drive.Status.SerialNumber = device.Serial
		updated = true
	}

	if drive.Status.FilesystemUUID != device.FSUUID {
		drive.Status.FilesystemUUID = device.FSUUID
		updated = true
	}

	if drive.Status.PartitionUUID != device.PartUUID {
		drive.Status.PartitionUUID = device.PartUUID
		updated = true
	}

	if drive.Status.MajorNumber != uint32(device.Major) {
		drive.Status.MajorNumber = uint32(device.Major)
		updated = true
	}

	if drive.Status.MinorNumber != uint32(device.Minor) {
		drive.Status.MinorNumber = uint32(device.Minor)
		updated = true
	}

	if drive.Status.UeventSerial != device.UeventSerial {
		drive.Status.UeventSerial = device.UeventSerial
		updated = true
	}

	if drive.Status.UeventFSUUID != device.UeventFSUUID {
		drive.Status.UeventFSUUID = device.UeventFSUUID
		updated = true
	}

	if drive.Status.WWID != device.WWID {
		drive.Status.WWID = device.WWID
		updated = true
	}

	if drive.Status.Vendor != device.Vendor {
		drive.Status.Vendor = device.Vendor
		updated = true
	}

	if drive.Status.DMName != device.DMName {
		drive.Status.DMName = device.DMName
		updated = true
	}

	if drive.Status.DMUUID != device.DMUUID {
		drive.Status.DMUUID = device.DMUUID
		updated = true
	}

	if drive.Status.MDUUID != device.MDUUID {
		drive.Status.MDUUID = device.MDUUID
		updated = true
	}

	if drive.Status.PartTableUUID != device.PTUUID {
		drive.Status.PartTableUUID = device.PTUUID
		updated = true
	}

	if !ptTypeEqual(drive.Status.PartTableType, device.PTType) {
		drive.Status.PartTableType = device.PTType
		updated = true
	}

	if drive.Status.Virtual != device.Virtual {
		drive.Status.Virtual = device.Virtual
		updated = true
	}

	if drive.Status.ReadOnly != device.ReadOnly {
		drive.Status.ReadOnly = device.ReadOnly
		updated = true
	}

	if drive.Status.Partitioned != device.Partitioned {
		drive.Status.Partitioned = device.Partitioned
		updated = true
	}

	if drive.Status.SwapOn != device.SwapOn {
		drive.Status.SwapOn = device.SwapOn
		updated = true
	}

	if drive.Status.Master != device.Master {
		drive.Status.Master = device.Master
		updated = true
	}

	return updated, nameChanged
}

func matchDrives(drives []directcsi.DirectCSIDrive, device *sys.Device) (indices []int) {
	for i, drive := range drives {
		switch {
		case isHWInfoAvailable(&drive):
			if matchDeviceHWInfo(&drive, device) {
				klog.Errorf("DEBUG:: matchDeviceHWInfo(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		case isDMMDUUIDAvailable(&drive):
			if matchDeviceDMMDUUID(&drive, device) {
				klog.Errorf("DEBUG:: matchDeviceDMMDUUID(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		case isPTUUIDAvailable(&drive):
			if matchDevicePTUUID(&drive, device) {
				klog.Errorf("DEBUG:: matchDevicePTUUID(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		case isPartUUIDAvailable(&drive):
			if matchDevicePartUUID(&drive, device) {
				klog.Errorf("DEBUG:: matchDevicePartUUID(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		case isFSUUIDAvailable(&drive):
			if matchDeviceFSUUID(&drive, device) {
				klog.Errorf("DEBUG:: matchDeviceFSUUID(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		case isV1Beta1Drive(&drive):
			if matchV1Beta1Name(&drive, device) {
				klog.Errorf("DEBUG:: matchV1Beta1Name(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		default:
			if matchDeviceNameSize(&drive, device) {
				klog.Errorf("DEBUG:: matchDeviceNameSize(): drive.Status.Path=%v, device.Name=%v", drive.Status.Path, device.Name)
				indices = append(indices, i)
			}
		}
	}

	return indices
}
