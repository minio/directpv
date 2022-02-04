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

const (
	// HostDevRoot is "/dev" directory.
	HostDevRoot = "/dev"

	// MountRoot is "/var/lib/direct-csi/mnt" directory.
	MountRoot = "/var/lib/direct-csi/mnt"

	// DirectCSIDevRoot is "/var/lib/direct-csi/devices" directory.
	DirectCSIDevRoot = "/var/lib/direct-csi/devices"

	// DirectCSIPartitionInfix is partition infix value.
	DirectCSIPartitionInfix = "-part-"

	// HostPartitionInfix is host infix value.
	HostPartitionInfix = "p"

	// MinSupportedDeviceSize is minimum supported size for default XFS filesystem.
	MinSupportedDeviceSize = 16 * 1024 * 1024 // 16 MiB

	runUdevData = "/run/udev/data"
)
