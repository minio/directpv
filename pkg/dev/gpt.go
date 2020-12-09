// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package dev

var GPTSignature = [8]byte{0x45, 0x46, 0x49, 0x20, 0x50, 0x41, 0x52, 0x54}

type GPTHeader struct {
	_                      [512]byte // LBA0: 512-byte MBR
	Signature              [8]byte   `json:"signature,omitempty"`
	Revision               [4]byte   `json:"revision,omitempty"`
	HeaderSize             uint32    `json:"HeaderSize,omitempty"`
	CRC32                  uint32    `json:"crc32,omitempty"`
	_                      uint32
	CurrentLBA             uint64   `json:"currentLBA,omitempty"`     // address of current LBA
	BackupLBA              uint64   `json:"backupLBA,omitempty"`      // address of backup LBA
	FirstUsableLBA         uint64   `json:"firstUsableLBA,omitempty"` // primary partition table last LBA + 1
	LastUsableLBA          uint64   `json:"lastUsableLBA,omitempty"`  // secondary parition table first LBA - 1
	DiskGUID               [16]byte `json:"diskGUID,omitempty"`
	PartitionEntryStartLBA uint64   `json:"partitionEntryStartLBA,omitempty"`
	NumPartitionEntries    uint32   `json:"numPartitionEntries,omitempty"`
	PartitionEntrySize     uint32   `json:"partitionEntrySize,omitempty"`
	PartitionArrayCRC32    uint32   `json:"partitionArrayCRC32,omitempty"`
}

func (g GPTHeader) Is() bool {
	for i := range GPTSignature {
		if GPTSignature[i] != g.Signature[i] {
			return false
		}
	}
	return true
}

type LBA struct {
	PartitionType [16]byte `json:"partitionType,omitempty"`
	PartitionGUID [16]byte `json:"partitionGUID,omitempty"`

	Start uint64 `json:"Start,omitempty"`
	End   uint64 `json:"End,omitempty"`
}

func (l LBA) Is() bool {
	invalidLBA := [16]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}
	for i := range invalidLBA {
		if l.PartitionType[i] != invalidLBA[i] {
			return true
		}
	}
	return false
}
