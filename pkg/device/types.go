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

package device

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/minio/directpv/pkg/xfs"
)

// Device is a block device information.
type Device struct {
	Name       string
	MajorMinor string

	// Populated from /sys
	Hidden      bool
	Removable   bool
	ReadOnly    bool
	Size        uint64
	Partitioned bool
	Holders     []string

	// Populated from /proc/1/mountinfo
	MountPoints []string

	// Populated by probing /proc/
	SwapOn bool
	CDRom  bool

	// populated by reading the device
	FSUUID        string
	TotalCapacity uint64
	FreeCapacity  uint64

	// Populated from /run/udev/data/b<Major>:<Minor>
	UDevData map[string]string
}

// DevPath return /dev notation of the path
func (d Device) Path() string {
	return path.Join("/dev", d.Name)
}

// FSType fetches the FSType value from the udevdata
func (d Device) FSType() string {
	if d.UDevData == nil {
		return ""
	}
	return d.UDevData["ID_FS_TYPE"]
}

// PartitionNumber fetches the paritionNumber from the udevData
func (d Device) PartitionNumber() (partition int, err error) {
	if d.UDevData == nil {
		err = fmt.Errorf("found nil udevdata for device %s", d.Name)
		return
	}
	if value, found := d.UDevData["ID_PART_ENTRY_NUMBER"]; found {
		partition, err = strconv.Atoi(value)
		if err != nil {
			return
		}
	}
	return
}

func (d Device) Model() string {
	if d.UDevData == nil {
		return ""
	}
	return d.UDevData["ID_MODEL"]
}

func (d Device) Vendor() string {
	if d.UDevData == nil {
		return ""
	}
	return d.UDevData["ID_VENDOR"]
}

func (d Device) IsUnavailable() (bool, string) {
	if d.Size < xfs.MinSupportedDeviceSize {
		return true, fmt.Sprintf("device size less than min supported %v", xfs.MinSupportedDeviceSize)
	}
	if d.SwapOn {
		return true, "device has swapOn enabled"
	}
	if d.Hidden {
		return true, "hidden device"
	}
	if d.ReadOnly {
		return true, "read-only device"
	}
	if d.Partitioned {
		return true, "partitioned device"
	}
	if len(d.Holders) > 0 {
		return true, "device has holders"
	}
	if len(d.MountPoints) > 0 {
		return true, "device is mounted"
	}
	if isLVMMemberFSType(d.FSType()) {
		return true, "device is a lvm member"
	}
	if d.CDRom {
		return true, "device is a CDROM"
	}
	return false, "available device"
}

func isLVMMemberFSType(fsType string) bool {
	return strings.EqualFold("LVM2_member", fsType)
}
