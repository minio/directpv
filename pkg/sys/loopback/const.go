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

package loopback

const (
	// DirectCSIBackFileRoot denotes loopback root.
	DirectCSIBackFileRoot = "/var/lib/direct-csi/loop"
	loopDeviceFormat      = "/dev/loop%d"
	loopControlPath       = "/dev/loop-control"
	nameSize              = 64
	keySize               = 32

	// Syscalls
	ctlAdd      = 0x4C80
	ctlRemove   = 0x4C81
	ctlGetFree  = 0x4C82
	setFD       = 0x4C00
	clrFD       = 0x4C01
	setStatus64 = 0x4C04
	getStatus64 = 0x4C05

	oneMB = 1048576
)
