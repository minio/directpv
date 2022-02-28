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

package uevent

import (
	"reflect"
	"strconv"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
)

func mapToUdevData(eventMap map[string]string) (*sys.UDevData, error) {
	path := eventMap["DEVPATH"]
	if path == "" {
		return nil, errInvalidDevPath
	}

	major, err := strconv.Atoi(eventMap["MAJOR"])
	if err != nil {
		return nil, err
	}

	minor, err := strconv.Atoi(eventMap["MINOR"])
	if err != nil {
		return nil, err
	}

	var partition int
	if value, found := eventMap["ID_PART_ENTRY_NUMBER"]; found && value != "" {
		partition, err = strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
	}

	return &sys.UDevData{
		Path:         path,
		Major:        major,
		Minor:        minor,
		Partition:    partition,
		WWID:         eventMap["ID_WWN"],
		Model:        eventMap["ID_MODEL"],
		UeventSerial: eventMap["ID_SERIAL_SHORT"],
		Vendor:       eventMap["ID_VENDOR"],
		DMName:       eventMap["DM_NAME"],
		DMUUID:       eventMap["DM_UUID"],
		MDUUID:       sys.NormalizeUUID(eventMap["MD_UUID"]),
		PTUUID:       eventMap["ID_PART_TABLE_UUID"],
		PTType:       eventMap["ID_PART_TABLE_TYPE"],
		PartUUID:     eventMap["ID_PART_ENTRY_UUID"],
		UeventFSUUID: eventMap["ID_FS_UUID"],
		FSType:       eventMap["ID_FS_TYPE"],
	}, nil
}

func validateNonHostInfo(directCSIDrive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	if directCSIDrive.Status.UeventFSUUID != device.UeventFSUUID {
		return false
	}
	if directCSIDrive.Status.Path != device.DevPath() {
		return false
	}
	if directCSIDrive.Status.MajorNumber != uint32(device.Major) {
		return false
	}
	if directCSIDrive.Status.MinorNumber != uint32(device.Minor) {
		return false
	}
	if directCSIDrive.Status.Virtual != device.Virtual {
		return false
	}
	if directCSIDrive.Status.PartitionNum != device.Partition {
		return false
	}
	if directCSIDrive.Status.WWID != device.WWID {
		return false
	}
	if directCSIDrive.Status.ModelNumber != device.Model {
		return false
	}
	if directCSIDrive.Status.UeventSerial != device.UeventSerial {
		return false
	}
	if directCSIDrive.Status.Vendor != device.Vendor {
		return false
	}
	if directCSIDrive.Status.DMName != device.DMName {
		return false
	}
	if directCSIDrive.Status.DMUUID != device.DMUUID {
		return false
	}
	if directCSIDrive.Status.MDUUID != device.MDUUID {
		return false
	}
	if directCSIDrive.Status.PartTableUUID != device.PTUUID {
		return false
	}
	if directCSIDrive.Status.PartTableType != device.PTType {
		return false
	}
	if directCSIDrive.Status.PartitionUUID != device.PartUUID {
		return false
	}
	if directCSIDrive.Status.Filesystem != device.FSType {
		return false
	}
	return true
}

func validateHostInfo(directCSIDrive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	if directCSIDrive.Status.Mountpoint != device.FirstMountPoint {
		return false
	}
	if !reflect.DeepEqual(directCSIDrive.Status.MountOptions, device.FirstMountOptions) {
		return false
	}
	if directCSIDrive.Status.FilesystemUUID != device.FSUUID {
		return false
	}
	if directCSIDrive.Status.Partitioned != device.Partitioned {
		return false
	}
	if directCSIDrive.Status.SwapOn != device.SwapOn {
		return false
	}
	if directCSIDrive.Status.Master != device.Master {
		return false
	}
	if directCSIDrive.Status.ReadOnly != device.ReadOnly {
		return false
	}
	if directCSIDrive.Status.RootPartition != device.Name {
		return false
	}
	if directCSIDrive.Status.SerialNumber != device.Serial {
		return false
	}
	if directCSIDrive.Status.AllocatedCapacity != int64(device.Size-device.FreeCapacity) {
		return false
	}
	if directCSIDrive.Status.FreeCapacity != int64(device.FreeCapacity) {
		return false
	}
	if directCSIDrive.Status.TotalCapacity != int64(device.Size) {
		return false
	}
	if directCSIDrive.Status.PhysicalBlockSize != int64(device.PhysicalBlockSize) {
		return false
	}
	if directCSIDrive.Status.LogicalBlockSize != int64(device.LogicalBlockSize) {
		return false
	}

	return true
}
