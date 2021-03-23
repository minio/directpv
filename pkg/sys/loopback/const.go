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
	LoopDeviceFormat      = "/dev/loop%d"
	LoopControlPath       = "/dev/loop-control"
	DirectCSIBackFileRoot = "/var/lib/direct-csi/loop"
	NameSize              = 64
	KeySize               = 32

	// Syscalls
	CtlAdd      = 0x4C80
	CtlRemove   = 0x4C81
	CtlGetFree  = 0x4C82
	SetFd       = 0x4C00
	ClrFd       = 0x4C01
	SetStatus   = 0x4C02
	SetStatus64 = 0x4C04
	GetStatus64 = 0x4C05

	oneMB = 1048576
)
