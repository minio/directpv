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
	"bytes"
	"encoding/binary"
	"fmt"
	simd "github.com/minio/sha256-simd"
	"os"
)

type FAT32 struct {
	SuperBlock   *FAT32SuperBlock
	capacityInfo *FAT32CapacityInfo
}

func NewFAT32() *FAT32 {
	return &FAT32{}
}

type FAT32SuperBlock struct {
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

type FAT32FSInformationSector struct {
	freeDataClustersCount uint32
	lastAllocatedCluster  uint32
}

type FAT32CapacityInfo struct {
	TotalCapacity uint64
	FreeCapacity  uint64
}

func (f32 *FAT32SuperBlock) Is() bool {
	trimmed := bytes.Trim(f32.Magic[:], " ")
	return bytes.Equal(trimmed, []byte(FAT32Magic))
}

func (f32 *FAT32) UUID() (string, error) {
	return fmt.Sprintf("%x", simd.Sum256([]byte(f32.SuperBlock.Serno[:]))), nil
}

func (f32 *FAT32) Type() string {
	return FSTypeFAT32
}

func (f32 *FAT32) FSBlockSize() uint64 {
	sectorSize := []byte(f32.SuperBlock.SectorSize[:])
	return uint64(f32.ByteOrder().Uint16(sectorSize))
}

func (f32 *FAT32) TotalCapacity() uint64 {
	return f32.capacityInfo.TotalCapacity
}

func (f32 *FAT32) FreeCapacity() uint64 {
	return f32.capacityInfo.FreeCapacity
}

func (f32 *FAT32) fsInformationSectorFromBytes(b []byte) (*FAT32FSInformationSector, error) {
	bLen := len(b)
	if bLen != 512 {
		return nil, fmt.Errorf("Cannot read FAT32 FS Information Sector from %d bytes instead of expected 512", bLen)
	}

	fsis := FAT32FSInformationSector{}

	// validate the signatures
	signatureStart := f32.ByteOrder().Uint32(b[0:4])
	signatureMid := f32.ByteOrder().Uint32(b[484:488])
	signatureEnd := f32.ByteOrder().Uint32(b[508:512])

	if signatureStart != FAT32FSInfoSectorSignatureStart {
		return nil, fmt.Errorf("Invalid signature at beginning of FAT 32 Filesystem Information Sector: %x", signatureStart)
	}
	if signatureMid != FAT32FSInfoSectorSignatureMid {
		return nil, fmt.Errorf("Invalid signature at middle of FAT 32 Filesystem Information Sector: %x", signatureMid)
	}
	if signatureEnd != FAT32FSInfoSectorSignatureEnd {
		return nil, fmt.Errorf("Invalid signature at end of FAT 32 Filesystem Information Sector: %x", signatureEnd)
	}

	fsis.freeDataClustersCount = f32.ByteOrder().Uint32(b[488:492])
	fsis.lastAllocatedCluster = f32.ByteOrder().Uint32(b[492:496])

	return &fsis, nil
}

func (f32 *FAT32) ByteOrder() binary.ByteOrder {
	return binary.LittleEndian
}

func (f32 *FAT32) ProbeFS(devicePath string, startOffset int64) (bool, error) {
	devFile, err := os.OpenFile(devicePath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return false, err
	}
	defer devFile.Close()

	if _, err = devFile.Seek(startOffset, os.SEEK_SET); err != nil {
		return false, err
	}

	f32SuperBlock := &FAT32SuperBlock{}
	if err := binary.Read(devFile, f32.ByteOrder(), f32SuperBlock); err != nil {
		return false, err
	}
	f32.SuperBlock = f32SuperBlock

	if !f32.SuperBlock.Is() {
		return false, nil
	}

	sectorSize := f32.FSBlockSize()
	fsisBytes := make([]byte, 512, 512)
	infoSectorBlock := int64(f32.SuperBlock.FsinfoSector) * int64(sectorSize)
	read, err := devFile.ReadAt(fsisBytes, startOffset+infoSectorBlock)
	if err != nil {
		return false, fmt.Errorf("Unable to read bytes for FSInformationSector: %v", err)
	}
	if read != 512 {
		return false, fmt.Errorf("Read %d bytes instead of expected %d for FS Information Sector", read, 512)
	}
	fsis, err := f32.fsInformationSectorFromBytes(fsisBytes)
	if err != nil {
		return false, fmt.Errorf("Error reading FileSystem Information Sector: %v", err)
	}

	freeCapacity := uint64(fsis.freeDataClustersCount) * uint64(f32.SuperBlock.ClusterSize) * sectorSize
	totalCapacity := uint64(f32.SuperBlock.TotalSect) * sectorSize

	f32.capacityInfo = &FAT32CapacityInfo{
		TotalCapacity: totalCapacity,
		FreeCapacity:  freeCapacity,
	}

	return true, nil
}
