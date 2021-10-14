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

package smart

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

//intSize is the size in bytes (converted to integer) of 0
const intSize int = int(unsafe.Sizeof(0))

// A ByteOrder specifies how to convert byte sequences
// into 16-, 32-, or 64-bit unsigned integers.
var (
	nativeEndian binary.ByteOrder
)

// init determines native endianness of a system
func init() {
	i := 0x1
	b := (*[intSize]byte)(unsafe.Pointer(&i))
	if b[0] == 1 {
		// LittleEndian is the little-endian implementation of ByteOrder
		nativeEndian = binary.LittleEndian
	} else {
		// BigEndian is the Big-endian implementation of ByteOrder
		nativeEndian = binary.BigEndian
	}
}

func sysIOCTL(fd, cmd, ptr uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, cmd, ptr)
	if errno != 0 {
		return errno
	}
	return nil
}

type smartDevice interface {
	SerialNumber() (string, error)
}

func getSmartDevice(devicePath string) smartDevice {
	if isNVMEDevice(devicePath) {
		return newNVMeDevice(devicePath)
	}
	return newSCSIDevice(devicePath)
}
