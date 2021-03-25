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

package mbr

var MBRSignature = uint16(0xaa55)

type ClassicMBRHeader struct {
	BootstrapCode    [446]byte       `json:"bootstrapCode,omitempty"`
	PartitionEntries [4]MBRPartition `json:"partitionEntries,omitempty"`
	BootSignature    uint16          `json:"signature,omitempty"`
}

type ModernStandardMBRHeader struct {
	BootStrapCode       [218]byte       `json:"bootstrapCode,omitempty"`
	Empty               uint16          `json:"-"`
	DiskTimestamp       DiskTimestamp   `json:"diskTimestamp,omitempty"`
	SecondBootStrapCode [216]byte       `json:"secondBootstrapCode,omitempty"`
	DiskSignature       uint32          `json:"signature"`
	CopyProtectedStatus uint16          `json:"copyProtectedStatus"`
	PartitionEntries    [4]MBRPartition `json:"partitionEntries,omitempty"`
	BootSignature       uint16          `json:"signature,omitempty"`
}

type MSDOSMBRHeader struct {
	BootStrapCode    [380]byte       `json:"bootstrapCode,omitempty"`
	MSDOSSignature   uint16          `json:"msdosSignature,omitempty"`
	PartitionEntries [8]MBRPartition `json:"partitionEntries,omitempty"`
	BootSignature    uint16          `json:"signature,omitempty"`
}

type AAPMBRHeader struct {
	BootStrapCode    [428]byte       `json:"bootstrapCode,omitempty"`
	AAPSignature     uint16          `json:"msdosSignature,omitempty"`
	AAPRecord        AAPRecord       `json:"aapRecord,omitempty"`
	PartitionEntries [4]MBRPartition `json:"partitionEntries,omitempty"`
	BootSignature    uint16          `json:"signature,omitempty"`
}

type AAPRecord struct {
	AAPPhysicalDrive uint8  `json:"aapPhysicalDrive,omitempty"`
	FirstCHS         CHS    `json:"firstCHS"`
	AAPPartitionType uint8  `json:"aapPartitionType"`
	LastCHS          CHS    `json:"lastCHS"`
	FirstLBA         uint32 `json:"firstLBA"`
	NumSectos        uint32 `json:"numSectors"`
}

type MBRPartition struct {
	Status        uint8  `json:"status"`
	FirstCHS      CHS    `json:"firstCHS"`
	PartitionType uint8  `json:"partitionType"`
	LastCHS       CHS    `json:"lastCHS"`
	FirstLBA      uint32 `json:"firstLBA"`
	NumSectors    uint32 `json:"numSectors"`
}

type CHS struct {
	Cylinder uint8 `json:"cylinder"`
	Head     uint8 `json:"head"`
	Sector   uint8 `json:"sector"`
}

type DiskTimestamp struct {
	OriginalPhysicalDrive uint8 `json:"originalPhysicalDrive"`
	Seconds               uint8 `json:"seconds"`
	Minutes               uint8 `json:"minutes"`
	Hours                 uint8 `json:"hours"`
}

func (c ClassicMBRHeader) Is() bool {
	return c.BootSignature == MBRSignature
}

func (m ModernStandardMBRHeader) Is() bool {
	if m.BootSignature != MBRSignature {
		return false
	}
	if m.Empty != uint16(0x0000) {
		return false
	}
	return true
}

func (m MSDOSMBRHeader) Is() bool {
	if m.BootSignature != MBRSignature {
		return false
	}
	if m.MSDOSSignature != uint16(0xa55a) {
		return false
	}
	return true
}

func (a AAPMBRHeader) Is() bool {
	if a.BootSignature != MBRSignature {
		return false
	}
	if a.AAPSignature != uint16(0x5678) {
		return false
	}
	return true
}

func (p MBRPartition) Is() bool {
	if p.Status == uint8(0x00) {
		return false
	}
	return true
}

func mbrPartType(partType uint8) string {
	return ""
}
