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

import "regexp"

var LoopRegexp = regexp.MustCompile("^loop[0-9]*")

// Device is a block device information.
type Device struct {
	// Populated from /sys
	Name      string
	Major     int
	Minor     int
	Removable bool
	ReadOnly  bool
	Virtual   bool
	Hidden    bool

	// Populated from /run/udev/data/b<Major>:<Minor>
	Size      uint64
	Partition int
	WWID      string
	Model     string
	Serial    string
	Vendor    string
	DMName    string
	DMUUID    string
	MDUUID    string
	PTUUID    string
	PTType    string
	PartUUID  string
	FSUUID    string
	FSType    string

	UeventSerial string
	UeventFSUUID string

	// Computed
	Parent      string
	Master      string
	Partitioned bool

	// Populated by reading device
	TotalCapacity     uint64
	FreeCapacity      uint64
	LogicalBlockSize  uint64
	PhysicalBlockSize uint64
	SwapOn            bool

	// Populated from /proc/1/mountinfo
	MountPoints       []string
	FirstMountPoint   string
	FirstMountOptions []string
}
