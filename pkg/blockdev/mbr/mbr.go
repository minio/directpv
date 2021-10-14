/*
 * This file is part of MinIO Direct CSI
 * Copyright (c) 2021 MinIO, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package mbr

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/minio/direct-csi/pkg/blockdev/parttable"
)

// ErrGPTProtectiveMBR denotes GPT protected MBR found error.
var ErrGPTProtectiveMBR = errors.New("GPT protective MBR found")

// MBR is interface compatible partition table information.
type MBR struct {
	partitions map[int]*parttable.Partition
}

// Type returns "msdos"
func (mbr *MBR) Type() string {
	return "msdos"
}

// UUID returns partition table UUID.
func (mbr *MBR) UUID() string {
	return ""
}

// Partitions returns list of partitions.
func (mbr *MBR) Partitions() map[int]*parttable.Partition {
	return mbr.partitions
}

// CHS denotes Cylinder-Head-Sector address.
type CHS struct {
	Cylinder uint8 // 1 byte.
	Head     uint8 // 1 byte.
	Sector   uint8 // 1 byte.
}

// PartEntry denotes partition entry.
type PartEntry struct {
	Status        uint8  // 1 byte
	FirstCHS      CHS    // 3 bytes.
	PartitionType uint8  // 1 byte.
	LastCHS       CHS    // 3 bytes.
	FirstLBA      uint32 // 4 bytes.
	NumSectors    uint32 // 4 bytes.
}

// MSDOSHeader denotes MSDOS MBR header.
type MSDOSHeader struct {
	BootstrapCode    [380]byte
	MSDOSSignature   uint16       // 2 bytes.
	PartitionEntries [8]PartEntry // 8 x 16 bytes.
	BootSignature    uint16       // 2 bytes.
}

// AAPHeader denotes Advanced Active Partitions MBR header.
type AAPHeader struct {
	BootstrapCode    [428]byte
	AAPSignature     uint16       // 2 bytes.
	AAPPhysicalDrive uint8        // 1 byte.
	FirstCHS         CHS          // 3 bytes.
	AAPPartitionType uint8        // 1 byte.
	LastCHS          CHS          // 3 bytes.
	FirstLBA         uint32       // 4 bytes.
	NumSectors       uint32       // 4 bytes.
	PartitionEntries [4]PartEntry // 4 x 16 bytes.
	BootSignature    uint16       // 2 bytes.
}

// ModernStandardHeader denotes modern standard MBR header.
type ModernStandardHeader struct {
	BootstrapCode         [218]byte
	Empty                 uint16 // 2 bytes.
	OriginalPhysicalDrive uint8  // 1 byte.
	Seconds               uint8  // 1 byte.
	Minutes               uint8  // 1 byte.
	Hours                 uint8  // 1 byte.
	SecondBootstrapCode   [216]byte
	DiskSignature         uint32       // 4 bytes.
	CopyProtectedStatus   uint16       // 2 bytes.
	PartitionEntries      [4]PartEntry // 4 x 16 bytes.
	BootSignature         uint16       // 2 bytes.
}

// ClassicHeader denotes classical generic MBR header.
type ClassicHeader struct {
	BootstrapCode    [446]byte
	PartitionEntries [4]PartEntry // 4 x 16 bytes.
	BootSignature    uint16       // 2 bytes.
}

func probeMSDOSMBR(data []byte) ([]PartEntry, error) {
	var header MSDOSHeader
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	if header.MSDOSSignature != 0xa55a {
		return nil, parttable.ErrPartTableNotFound
	}

	return header.PartitionEntries[:], nil
}

func probeAAPMBR(data []byte) ([]PartEntry, error) {
	var header AAPHeader
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.AAPSignature != 0x5678 {
		return nil, parttable.ErrPartTableNotFound
	}

	return header.PartitionEntries[:], nil
}

func probeModernStandardMBR(data []byte) ([]PartEntry, error) {
	var header ModernStandardHeader
	err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	if header.Empty != 0 {
		return nil, parttable.ErrPartTableNotFound
	}

	if header.PartitionEntries[0].PartitionType == 0xEE {
		return nil, ErrGPTProtectiveMBR
	}

	return header.PartitionEntries[:], nil
}

func probeClassicMBR(data []byte) ([]PartEntry, error) {
	var header ClassicHeader
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	return header.PartitionEntries[:], nil
}

func probe(data []byte) (partEntries []PartEntry, err error) {
	if !bytes.HasSuffix(data, []byte{0x55, 0xAA}) {
		return nil, parttable.ErrPartTableNotFound
	}

	if partEntries, err = probeMSDOSMBR(data); !errors.Is(err, parttable.ErrPartTableNotFound) {
		return
	}

	if partEntries, err = probeAAPMBR(data); !errors.Is(err, parttable.ErrPartTableNotFound) {
		return
	}

	if partEntries, err = probeModernStandardMBR(data); !errors.Is(err, parttable.ErrPartTableNotFound) {
		return
	}

	return probeClassicMBR(data)
}

// Probe reads and returns MBR style partition table.
func Probe(readSeeker io.ReadSeeker) (mbr *MBR, err error) {
	data := make([]byte, 512)
	read := func() {
		if _, err = io.ReadFull(readSeeker, data); err != nil {
			return
		}
	}

	if read(); err != nil {
		return nil, err
	}
	partEntries, err := probe(data)
	if err != nil {
		return nil, err
	}

	partitions := map[int]*parttable.Partition{}
	add := func(n int, entry PartEntry, partType parttable.PartType) {
		if entry.PartitionType != 0 {
			partitions[n] = &parttable.Partition{
				Number: n,
				Type:   partType,
			}
		}
	}

	for i, entry := range partEntries {
		switch entry.PartitionType {
		case 0x05, 0x0F, 0x85, 0xC5, 0xCF, 0xD5:
			add(i+1, entry, parttable.Extended)
			if _, err = readSeeker.Seek(int64(entry.FirstLBA-1)*512, os.SEEK_CUR); err != nil {
				return nil, err
			}
			if read(); err != nil {
				return nil, err
			}
			logicalPartEntries, err := probe(data)
			if err != nil {
				return nil, err
			}
			for j := range logicalPartEntries {
				add(j+5, logicalPartEntries[j], parttable.Logical)
			}
		default:
			add(i+1, entry, parttable.Primary)
		}
	}

	return &MBR{partitions: partitions}, nil
}
