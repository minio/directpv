// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"bytes"
	"encoding/binary"
	"strings"
	"unsafe"

	"github.com/dswarbrick/smart/ioctl"
	"golang.org/x/sys/unix"
)

const (
	nvmeAdminIdentify = 0x06
)

var (
	nvmeIoctlAdminCmd = ioctl.Iowr('N', 0x41, unsafe.Sizeof(nvmePassthruCommand{}))
)

type nvmeDevice struct {
	Name string
}

func isNVMEDevice(devPath string) bool {
	return strings.HasPrefix(devPath, "/dev/nvme")
}

func newNVMeDevice(name string) *nvmeDevice {
	return &nvmeDevice{name}
}

type nvmePassthruCommand struct {
	opcode  uint8  // opcode
	_       uint8  // flags
	_       uint16 // rsvd1
	nsid    uint32 // nsid
	_       uint32 // cdw2
	_       uint32 // cdw3
	_       uint64 // metadata
	addr    uint64 // addr
	_       uint32 // metadataLen
	dataLen uint32 // dataLen
	cdw10   uint32 // cdw10
	_       uint32 // cdw11
	_       uint32 // cdw12
	_       uint32 // cdw13
	_       uint32 // cdw14
	_       uint32 // cdw15
	_       uint32 // timeoutMS
	_       uint32 // result
} // 72 bytes

type nvmeIdentController struct {
	VendorID     uint16     // PCI Vendor ID
	Ssvid        uint16     // PCI Subsystem Vendor ID
	SerialNumber [20]byte   // Serial Number
	ModelNumber  [40]byte   // Model Number
	Firmware     [8]byte    // Firmware Revision
	Rab          uint8      // Recommended Arbitration Burst
	IEEE         [3]byte    // IEEE OUI Identifier
	Cmic         uint8      // Controller Multi-Path I/O and Namespace Sharing Capabilities
	Mdts         uint8      // Maximum Data Transfer Size
	Cntlid       uint16     // Controller ID
	Ver          uint32     // Version
	Rtd3r        uint32     // RTD3 Resume Latency
	Rtd3e        uint32     // RTD3 Entry Latency
	Oaes         uint32     // Optional Asynchronous Events Supported
	Rsvd96       [160]byte  // ...
	Oacs         uint16     // Optional Admin Command Support
	ACL          uint8      // Abort Command Limit
	Aerl         uint8      // Asynchronous Event Request Limit
	Frmw         uint8      // Firmware Updates
	Lpa          uint8      // Log Page Attributes
	Elpe         uint8      // Error Log Page Entries
	Npss         uint8      // Number of Power States Support
	Avscc        uint8      // Admin Vendor Specific Command Configuration
	Apsta        uint8      // Autonomous Power State Transition Attributes
	Wctemp       uint16     // Warning Composite Temperature Threshold
	Cctemp       uint16     // Critical Composite Temperature Threshold
	Mtfa         uint16     // Maximum Time for Firmware Activation
	Hmpre        uint32     // Host Memory Buffer Preferred Size
	Hmmin        uint32     // Host Memory Buffer Minimum Size
	Tnvmcap      [16]byte   // Total NVM Capacity
	Unvmcap      [16]byte   // Unallocated NVM Capacity
	Rpmbs        uint32     // Replay Protected Memory Block Support
	Rsvd316      [196]byte  // ...
	Sqes         uint8      // Submission Queue Entry Size
	Cqes         uint8      // Completion Queue Entry Size
	Rsvd514      [2]byte    // (defined in NVMe 1.3 spec)
	Nn           uint32     // Number of Namespaces
	Oncs         uint16     // Optional NVM Command Support
	Fuses        uint16     // Fused Operation Support
	Fna          uint8      // Format NVM Attributes
	Vwc          uint8      // Volatile Write Cache
	Awun         uint16     // Atomic Write Unit Normal
	Awupf        uint16     // Atomic Write Unit Power Fail
	Nvscc        uint8      // NVM Vendor Specific Command Configuration
	Rsvd531      uint8      // ...
	Acwu         uint16     // Atomic Compare & Write Unit
	Rsvd534      [2]byte    // ...
	Sgls         uint32     // SGL Support
	Rsvd540      [1508]byte // ...
	// Psd          [32]nvmeIdentPowerState // Power State Descriptors
	// Vs           [1024]byte              // Vendor Specific
} // 4096 bytes

func (d *nvmeDevice) open() (int, error) {
	return unix.Open(d.Name, unix.O_RDWR, 0600)
}

func (d *nvmeDevice) close(fd int) error {
	return unix.Close(fd)
}

func (d *nvmeDevice) SerialNumber() (string, error) {

	fd, err := d.open()
	if err != nil {
		return "", err
	}
	defer d.close(fd)

	buf := make([]byte, 4096)

	cmd := nvmePassthruCommand{
		opcode:  nvmeAdminIdentify,
		nsid:    0, // Namespace 0, since we are identifying the controller
		addr:    uint64(uintptr(unsafe.Pointer(&buf[0]))),
		dataLen: uint32(len(buf)),
		cdw10:   1, // Identify controller
	}

	if err := sysIOCTL(uintptr(fd), nvmeIoctlAdminCmd, uintptr(unsafe.Pointer(&cmd))); err != nil {
		return "", err
	}

	var controller nvmeIdentController

	if err := binary.Read(bytes.NewBuffer(buf[:]), nativeEndian, &controller); err != nil {
		return "", err
	}

	return string(controller.SerialNumber[:]), nil
}
