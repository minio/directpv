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

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"strconv"

	"github.com/golang/glog"
)

var (
	ErrNotGPT               = errors.New("Not a GPR partition")
	ErrNotClassicMBR        = errors.New("Not a Classic MBR partition")
	ErrNotModernStandardMBR = errors.New("Not a Modern Standard MBR partition")
	ErrNotAAPMBR            = errors.New("Not a AAP MBR partition")
)

func (b *BlockDevice) probeAAPMBR(ctx context.Context) ([]Partition, error) {
	devPath := getBlockFile(b.Devname)
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &AAPMBRHeader{}
	err = binary.Read(devFile, binary.LittleEndian, mbr)
	if err != nil {
		return nil, err
	}

	if !mbr.Is() {
		return nil, ErrNotAAPMBR
	}

	partitions := []Partition{}
	for i, p := range mbr.PartitionEntries {
		if !p.Is() {
			continue
		}
		partNum := int(i + 1)
		partitionPath := b.Path + "-part-" + strconv.Itoa(partNum)
		if err := makeBlockFile(partitionPath, b.Major, b.Minor); err != nil {
			return nil, err
		}

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
			},
			PartitionNum: uint32(partNum),
			Type:         mbrPartType(p.PartitionType),
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeClassicMBR(ctx context.Context) ([]Partition, error) {
	devPath := getBlockFile(b.Devname)
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &ClassicMBRHeader{}
	err = binary.Read(devFile, binary.LittleEndian, mbr)
	if err != nil {
		return nil, err
	}

	if !mbr.Is() {
		return nil, ErrNotClassicMBR
	}

	partitions := []Partition{}
	for i, p := range mbr.PartitionEntries {
		if !p.Is() {
			continue
		}
		partNum := int(i + 1)
		partitionPath := b.Path + "-part-" + strconv.Itoa(partNum)
		if err := makeBlockFile(partitionPath, b.Major, b.Minor); err != nil {
			return nil, err
		}

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
			},
			PartitionNum: uint32(partNum),
			Type:         mbrPartType(p.PartitionType),
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeModernStandardMBR(ctx context.Context) ([]Partition, error) {
	devPath := getBlockFile(b.Devname)
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &ModernStandardMBRHeader{}
	err = binary.Read(devFile, binary.LittleEndian, mbr)
	if err != nil {
		return nil, err
	}

	if !mbr.Is() {
		return nil, ErrNotModernStandardMBR
	}

	partitions := []Partition{}
	for i, p := range mbr.PartitionEntries {
		if !p.Is() {
			continue
		}
		partNum := int(i + 1)
		partitionPath := b.Path + "-part-" + strconv.Itoa(partNum)
		if err := makeBlockFile(partitionPath, b.Major, b.Minor); err != nil {
			return nil, err
		}

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
			},
			PartitionNum: uint32(partNum),
			Type:         mbrPartType(p.PartitionType),
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeGPT(ctx context.Context) ([]Partition, error) {
	devPath := getBlockFile(b.Devname)
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	gpt := &GPTHeader{}
	err = binary.Read(devFile, binary.LittleEndian, gpt)
	if err != nil {
		return nil, err
	}

	if !gpt.Is() {
		return nil, ErrNotGPT
	}

	// Skip 420 bytes of reserved space
	_, err = devFile.Seek(int64(420), os.SEEK_CUR)
	if err != nil {
		glog.Errorf("GPT header is corrupt")
		return nil, err
	}

	var offset int64
	pSize := gpt.PartitionEntrySize
	lbaSize := 48 // manually calculated based on struct definition
	if int64(pSize) > int64(lbaSize) {
		offset = int64(pSize) - int64(lbaSize)
	}

	partitions := []Partition{}
	for i := uint32(0); i < gpt.NumPartitionEntries; i++ {
		lba := &GPTLBA{}
		err = binary.Read(devFile, binary.LittleEndian, lba)
		if err != nil {
			return nil, err
		}

		_, err := devFile.Seek(offset, os.SEEK_CUR)
		if err != nil {
			glog.Errorf("LBA data is corrupt")
			return nil, err
		}

		// Only true after all the valid partition entries have been read
		if !lba.Is() {
			break
		}

		partTypeUUID := stringifyUUID(lba.PartitionType)
		partType := PartitionTypes[partTypeUUID]
		if partType == "" {
			partType = partTypeUUID
		}

		partNum := int(i + 1)
		partitionPath := b.Path + "-part-" + strconv.Itoa(partNum)
		if err := makeBlockFile(partitionPath, b.Major, b.Minor); err != nil {
			return nil, err
		}

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        lba.Start,
				EndBlock:          lba.End,
				TotalCapacity:     (lba.End - lba.Start) * b.LogicalBlockSize,
				NumBlocks:         lba.End - lba.Start,
				Path:              partitionPath,
			},
			PartitionNum:  uint32(partNum),
			Type:          partType,
			TypeUUID:      partTypeUUID,
			PartitionGUID: stringifyUUID(lba.PartitionGUID),
			DiskGUID:      stringifyUUID(gpt.DiskGUID),
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}
