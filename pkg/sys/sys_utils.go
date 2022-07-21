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

package sys

import (
	"fmt"
	"strings"
)

func isLVMMemberFSType(fsType string) bool {
	return strings.EqualFold("LVM2_member", fsType)
}

// NormalizeUUID normalizes the UUID
func NormalizeUUID(uuid string) string {
	if u := strings.ReplaceAll(strings.ReplaceAll(uuid, ":", ""), "-", ""); len(u) > 20 {
		uuid = fmt.Sprintf("%v-%v-%v-%v-%v", u[:8], u[8:12], u[12:16], u[16:20], u[20:])
	}
	return uuid
}

// IsDeviceUnavailable checks if the device is unavailable to use
func IsDeviceUnavailable(device *Device) bool {
	return device.Size < MinSupportedDeviceSize ||
		device.SwapOn ||
		device.Hidden ||
		device.ReadOnly ||
		device.Partitioned ||
		device.Master != "" ||
		len(device.Holders) > 0 ||
		device.FirstMountPoint != "" ||
		isLVMMemberFSType(device.FSType) ||
		device.CDRom
}
