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

package fat32

import (
	"encoding/binary"
	"fmt"
	"io"

	fserrors "github.com/minio/directpv/pkg/fs/errors"
)

// EBPB is FAT32 Extended BIOS Parameter Block.
type EBPB struct {
	Ignored     [3]uint8
	Sysid       [8]uint8 // OEM name/version
	SectorSize  [2]uint8 // Number of bytes per sector
	ClusterSize uint8    // Number of sectors per cluster
	Reserved    uint16   //  Number of reserved sectors
	Fats        uint8    // Number of FAT copies
	DirEntries  [2]uint8 // Number of root directory entries
	Sectors     [2]uint8 // Total number of sectors in the filesystem
	Media       uint8    //  Media descriptor type
	FatLength   uint16   // Number of sectors per FAT
	SecsTrack   uint16   // Number of sectors per track
	Heads       uint16   // Number of heads
	Hidden      uint32   // Number of hidden sectors
	/* BIOS Parameter Block ends here */
	// Bootstrap
	TotalSect    uint32
	Fat32Length  uint32
	Flags        uint16
	Version      [2]uint8
	RootCluster  uint32
	FsinfoSector uint16
	BackupBoot   uint16
	Reserved2    [6]uint16
	Unknown      [3]uint8
	Serno        [4]uint8
	Label        [11]uint8
	Magic        [8]uint8
	Dummy2       [0x1fe - 0x5a]uint8
	Pmagic       [2]uint8
}

// ReadEBPB reads FAT32 Extended BIOS Parameter Block.
func ReadEBPB(reader io.Reader) (*EBPB, error) {
	var ebpb EBPB
	if err := binary.Read(reader, binary.LittleEndian, &ebpb); err != nil {
		return nil, err
	}

	if string(ebpb.Magic[:]) != "FAT32   " {
		return nil, fserrors.ErrFSNotFound
	}

	return &ebpb, nil
}

// FAT32 contains filesystem information.
type FAT32 struct {
	id            string
	totalCapacity uint64
	freeCapacity  uint64
}

// ID returns filesystem UUID.
func (f *FAT32) ID() string {
	return f.id
}

// Type returns "fat32".
func (f *FAT32) Type() string {
	return "fat32"
}

// TotalCapacity returns total capacity of filesystem.
func (f *FAT32) TotalCapacity() uint64 {
	return f.totalCapacity
}

// FreeCapacity returns free capacity of filesystem.
func (f *FAT32) FreeCapacity() uint64 {
	return f.freeCapacity
}

// Probe tries to probe FAT32 superblock.
func Probe(reader io.ReadSeeker) (*FAT32, error) {
	ebpb, err := ReadEBPB(reader)
	if err != nil {
		return nil, err
	}

	blockSize := binary.LittleEndian.Uint16([]byte(ebpb.SectorSize[:]))
	if _, err = reader.Seek(int64(ebpb.FsinfoSector*blockSize), io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, 512)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	if binary.LittleEndian.Uint32(data[0:4]) != 0x41615252 {
		return nil, fmt.Errorf("start signature mismatch; expected=%v, got=%v", 0x41615252, binary.LittleEndian.Uint32(data[0:4]))
	}

	if binary.LittleEndian.Uint32(data[484:488]) != 0x61417272 {
		return nil, fmt.Errorf("middle signature mismatch; expected=%v, got=%v", 0x61417272, binary.LittleEndian.Uint32(data[484:488]))
	}

	if binary.LittleEndian.Uint32(data[508:512]) != 0xAA550000 {
		return nil, fmt.Errorf("end signature mismatch; expected=%v, got=%v", 0xAA550000, binary.LittleEndian.Uint32(data[508:512]))
	}

	freeClusters := binary.LittleEndian.Uint32(data[488:492])
	return &FAT32{
		id:            fmt.Sprintf("%04X-%04X", binary.LittleEndian.Uint16(ebpb.Serno[2:4]), binary.LittleEndian.Uint16(ebpb.Serno[0:2])),
		totalCapacity: uint64(ebpb.TotalSect) * uint64(blockSize),
		freeCapacity:  uint64(freeClusters) * uint64(ebpb.ClusterSize) * uint64(blockSize),
	}, nil
}
