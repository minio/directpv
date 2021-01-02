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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var (
	ErrNotPartition = errors.New("Not a partitioned volume")
)

type Partition struct {
	PartitionNum  uint32 `json:"partitionNum,omitempty"`
	Type          string `json:"partitionType,omitempty"`
	TypeUUID      string `json:"partitionTypeUUID,omitempty"`
	PartitionGUID string `json:"partitionGUID,omitempty"`
	DiskGUID      string `json:"diskGUID,omitempty"`

	*DriveInfo `json:"driveInfo,omitempty"`
}

func (b *BlockDevice) findPartitions(ctx context.Context) ([]Partition, error) {
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

func makeBlockFile(path string, major, minor uint32) error {
	if err := unix.Mknod(path, unix.S_IFBLK|uint32(os.FileMode(0666)), int(unix.Mkdev(major, minor))); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func getBlockFile(devName string) string {
	return filepath.Join(DevRoot, devName)
}
