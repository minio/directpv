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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

var ErrNotGPT = errors.New("Not a GPT volume")

type Partition struct {
	PartitionNum  uint32 `json:"partitionNum,omitempty"`
	Type          string `json:"partitionType,omitempty"`
	TypeUUID      string `json:"partitionTypeUUID,omitempty"`
	PartitionGUID string `json:"partitionGUID,omitempty"`
	DiskGUID      string `json:"diskGUID,omitempty"`

	*DriveInfo `json:"driveInfo,omitempty"`
}

func (b *BlockDevice) FindPartitions(ctx context.Context) ([]Partition, error) {
	devPath := filepath.Join("/dev/", b.Devname)
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

	partPaths, pErr := fetchPartitionPaths(b.Path)
	if pErr != nil {
		return nil, pErr
	}

	partitions := []Partition{}
	for i := uint32(0); i < gpt.NumPartitionEntries; i++ {
		lba := &LBA{}
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
		partitionPath := GetPartitionPath(partPaths, partNum)
		if partitionPath == "" {
			mkPath := b.Path + "p" + strconv.Itoa(partNum)
			if err := makeBlockFile(ctx, mkPath, b.Major, b.Minor); err != nil {
				return nil, err
			}
			partitionPath = mkPath
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

func makeBlockFile(ctx context.Context, path string, major, minor uint32) error {
	args := []string{
		path,
		"b",
		fmt.Sprint(major),
		fmt.Sprint(minor),
	}
	output, err := exec.CommandContext(ctx, "mknod", args...).CombinedOutput()
	if err != nil {
		log.Printf("Failed to mknod: err: (%s) output: (%s)", err.Error(), string(output))
	}
	return err
}

func fetchPartitionPaths(devPath string) ([]string, error) {
	partPaths := []string{}

	devDir, err := os.Open("/dev")
	defer devDir.Close()
	if err != nil {
		return partPaths, err
	}

	dentries, dErr := devDir.Readdirnames(0)
	if dErr != nil {
		return partPaths, dErr
	}

	for _, dentry := range dentries {
		absPath := filepath.Join("/dev", dentry)
		if absPath == devPath {
			continue
		}
		if strings.HasPrefix(absPath, devPath) {
			partPaths = append(partPaths, absPath)
		}
	}

	return partPaths, nil
}

func GetPartitionPath(partPaths []string, partNum int) string {
	for _, partPath := range partPaths {
		pNum, err := getPartNumFromPath(partPath)
		if err == nil {
			if pNum == partNum {
				return partPath
			}
		}
	}
	return ""
}

func getPartNumFromPath(partPath string) (int, error) {
	numStr := ""
	for i := len(partPath) - 1; i >= 0; i-- {
		_, err := strconv.Atoi(string(partPath[i]))
		if err != nil {
			if numStr != "" {
				return strconv.Atoi(numStr)
			}
			break
		}
		numStr = string(partPath[i]) + numStr
	}
	return 0, nil
}
