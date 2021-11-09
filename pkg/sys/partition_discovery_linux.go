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

package sys

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"

	"github.com/minio/direct-csi/pkg/sys/gpt"
	"github.com/minio/direct-csi/pkg/sys/mbr"
)

func (b *BlockDevice) probePartitions(ctx context.Context) ([]Partition, error) {
	parts, err := b.probeGPT(ctx)
	if err != nil {
		if err != ErrNotGPT {
			return nil, err
		}
	} else {
		return parts, nil
	}

	parts, err = b.probeAAPMBR(ctx)
	if err != nil {
		if err != ErrNotAAPMBR {
			return nil, err
		}
	} else {
		return parts, nil
	}

	parts, err = b.probeModernStandardMBR(ctx)
	if err != nil {
		if err != ErrNotModernStandardMBR {
			return nil, err
		}
	} else {
		return parts, nil
	}

	// This should be the last MBR check
	parts, err = b.probeClassicMBR(ctx)
	if err != nil {
		if err != ErrNotClassicMBR {
			return nil, err
		}
	} else {
		return parts, nil
	}

	return nil, ErrNotPartition
}

func (b *BlockDevice) probeAAPMBR(ctx context.Context) ([]Partition, error) {
	devPath := b.HostDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &mbr.AAPMBRHeader{}
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
		ueventInfo, partNum, err := parseUevent(partitionUeventPath(b.Devname, uint32(i+1)))
		if err != nil {
			klog.V(5).Info(err)
			return nil, err
		}
		partitionPath := fmt.Sprintf("%s%s%d", b.Path, DirectCSIPartitionInfix, i+1)

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
				Major:             ueventInfo.Major,
				Minor:             ueventInfo.Minor,
			},
			PartitionNum: partNum,
			// Type:          p.PartitionType,
			// TypeUUID:      "",
			// PartitionGUID: "",
			// DiskGUID:      "",
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeClassicMBR(ctx context.Context) ([]Partition, error) {
	devPath := b.HostDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &mbr.ClassicMBRHeader{}
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
		ueventInfo, partNum, err := parseUevent(partitionUeventPath(b.Devname, uint32(i+1)))
		if err != nil {
			klog.V(5).Info(err)
			return nil, err
		}
		partitionPath := fmt.Sprintf("%s%s%d", b.Path, DirectCSIPartitionInfix, i+1)

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
				Major:             ueventInfo.Major,
				Minor:             ueventInfo.Minor,
			},
			PartitionNum: partNum,
			// Type:          p.PartitionType,
			// TypeUUID:      "",
			// PartitionGUID: "",
			// DiskGUID:      "",
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeModernStandardMBR(ctx context.Context) ([]Partition, error) {
	devPath := b.HostDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbr := &mbr.ModernStandardMBRHeader{}
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
		ueventInfo, partNum, err := parseUevent(partitionUeventPath(b.Devname, uint32(i+1)))
		if err != nil {
			klog.V(5).Info(err)
			return nil, err
		}
		partitionPath := fmt.Sprintf("%s%s%d", b.Path, DirectCSIPartitionInfix, i+1)

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        uint64(p.FirstLBA),
				EndBlock:          uint64(p.FirstLBA) + (b.LogicalBlockSize * uint64(p.NumSectors)),
				TotalCapacity:     b.LogicalBlockSize * uint64(p.NumSectors),
				NumBlocks:         uint64(p.NumSectors),
				Path:              partitionPath,
				Major:             ueventInfo.Major,
				Minor:             ueventInfo.Minor,
			},
			PartitionNum: partNum,
			// Type:          p.PartitionType,
			// TypeUUID:      "",
			// PartitionGUID: "",
			// DiskGUID:      "",
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func (b *BlockDevice) probeGPT(ctx context.Context) ([]Partition, error) {
	devPath := b.HostDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	gptPart := &gpt.GPTHeader{}
	err = binary.Read(devFile, binary.LittleEndian, gptPart)
	if err != nil {
		return nil, err
	}

	if !gptPart.Is() {
		return nil, ErrNotGPT
	}

	// Skip 420 bytes of reserved space
	_, err = devFile.Seek(int64(420), os.SEEK_CUR)
	if err != nil {
		klog.Errorf("GPT header is corrupt")
		return nil, err
	}

	var offset int64
	pSize := gptPart.PartitionEntrySize
	lbaSize := 48 // manually calculated based on struct definition
	if int64(pSize) > int64(lbaSize) {
		offset = int64(pSize) - int64(lbaSize)
	}

	partitions := []Partition{}
	for i := uint32(0); i < gptPart.NumPartitionEntries; i++ {
		lba := &gpt.GPTLBA{}
		err = binary.Read(devFile, binary.LittleEndian, lba)
		if err != nil {
			return nil, err
		}

		_, err := devFile.Seek(offset, os.SEEK_CUR)
		if err != nil {
			klog.Errorf("LBA data is corrupt")
			return nil, err
		}

		// Only true after all the valid partition entries have been read
		if !lba.Is() {
			break
		}

		partTypeUUID := stringifyUUID(lba.PartitionType)
		partType := gpt.PartitionTypes[partTypeUUID]
		if partType == "" {
			partType = partTypeUUID
		}
		ueventInfo, partNum, err := parseUevent(partitionUeventPath(b.Devname, uint32(i+1)))
		if err != nil {
			klog.V(5).Info(err)
			return nil, err
		}

		partitionPath := fmt.Sprintf("%s%s%d", b.Path, DirectCSIPartitionInfix, i+1)

		part := Partition{
			DriveInfo: &DriveInfo{
				LogicalBlockSize:  b.LogicalBlockSize,
				PhysicalBlockSize: b.PhysicalBlockSize,
				StartBlock:        lba.Start,
				EndBlock:          lba.End,
				TotalCapacity:     (lba.End - lba.Start) * b.LogicalBlockSize,
				NumBlocks:         lba.End - lba.Start,
				Path:              partitionPath,
				Major:             ueventInfo.Major,
				Minor:             ueventInfo.Minor,
			},
			PartitionNum:  partNum,
			Type:          partType,
			TypeUUID:      partTypeUUID,
			PartitionGUID: stringifyUUID(lba.PartitionGUID),
			DiskGUID:      stringifyUUID(gptPart.DiskGUID),
		}
		partitions = append(partitions, part)
	}
	return partitions, nil
}

func partitionUeventPath(rootDevName string, partNum uint32) string {
	partSubPath := fmt.Sprintf("%s%d", rootDevName, partNum)
	if strings.HasPrefix(rootDevName, "nvme") {
		partSubPath = fmt.Sprintf("%sp%d", rootDevName, partNum)
	}

	return filepath.Join(
		"/sys",
		"block",
		rootDevName,
		partSubPath,
		"uevent",
	)
}

func stringifyUUID(uuid [16]byte) string {
	str := ""

	// first part of uuid is LittleEndian encoded uint32
	for i := 0; i < 4; i++ {
		str = str + fmt.Sprintf("%02X", uint8(uuid[3-i]))
	}
	str = str + "-"

	// next 2 bytes are LitteEndian encoded uint8
	str = str + fmt.Sprintf("%02X", uint8(uuid[5]))
	str = str + fmt.Sprintf("%02X", uint8(uuid[4]))
	str = str + "-"

	// next 2 bytes are LitteEndian encoded uint8
	str = str + fmt.Sprintf("%02X", uint8(uuid[7]))
	str = str + fmt.Sprintf("%02X", uint8(uuid[6]))
	str = str + "-"

	// rest should be taken in order
	for i := 8; i < 10; i++ {
		str = str + fmt.Sprintf("%02X", uint8(uuid[i]))
	}
	str = str + "-"

	for i := 10; i < 16; i++ {
		str = str + fmt.Sprintf("%02X", uint8(uuid[i]))
	}

	return str
}

func curr(f *os.File) int64 {
	offset, err := f.Seek(0, os.SEEK_CUR)
	if err == nil {
		return offset
	}
	return 0
}

func MakeBlockFile(path string, major, minor uint32) error {

	updateBlockFile := func(directCSIDevicePath string, major, minor uint32) error {
		majorN, minorN, err := GetMajorMinor(directCSIDevicePath)
		if err != nil {
			return err
		}
		if majorN == major && minorN == minor {
			// No change in (major, minor) pair
			return nil
		}

		// remove and a create a new device with correct (major, minor) pair
		if err := os.Remove(directCSIDevicePath); err != nil {
			return err
		}
		if err := createBlockFile(directCSIDevicePath, major, minor); err != nil {
			return err
		}
		return nil
	}

	if err := createBlockFile(path, major, minor); err != nil {
		if !os.IsExist(err) {
			return err
		}
		if err := updateBlockFile(path, major, minor); err != nil {
			return err
		}
	}
	return nil
}

func createBlockFile(path string, major, minor uint32) error {
	mkdevResp := unix.Mkdev(major, minor)
	if err := unix.Mknod(path, unix.S_IFBLK|uint32(os.FileMode(0666)), int(mkdevResp)); err != nil {
		return err
	}
	return nil
}
