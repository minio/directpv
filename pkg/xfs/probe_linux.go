//go:build linux

// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

package xfs

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"k8s.io/klog/v2"
)

func bytesToUUIDString(uuid [16]byte) string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%x-%x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		uuid[8:10],
		uuid[10:],
	)
}

type superBlock struct {
	MagicNumber         uint32
	BlockSize           uint32
	TotalBlocks         uint64
	RBlocks             uint64
	RExtents            uint64
	UUID                [16]byte
	FirstBlock          uint64
	RootInode           uint64
	ExtentsBitmapInode  uint64
	BitmapSummaryInode  uint64
	ExtentSize          uint32
	AGSize              uint32
	AGCount             uint32
	BitmapBlocks        uint32
	JournalBlocks       uint32
	FilesystemVersion   uint16
	SectorSize          uint16
	InodeSize           uint16
	Inodes              uint16
	FilesystemName      [12]byte
	LogBlockSize        uint8
	LogSectorSize       uint8
	LogInodeSize        uint8
	LogInodeOrBlockSize uint8
	LogAGSize           uint8
	LogExtents          uint8
	InProgress          uint8
	MaxInodePercentage  uint8
	AllocatedInodes     uint64
	FreeInodes          uint64
	FreeBlocks          uint64
	FreeExtents         uint64
	// Ignoring the rest
}

func readSuperBlock(reader io.Reader) (fsuuid, label string, totalCapacity, freeCapacity uint64, err error) {
	var sb superBlock
	if err = binary.Read(reader, binary.BigEndian, &sb); err != nil {
		return
	}

	if sb.MagicNumber == 0x58465342 {
		fsuuid = bytesToUUIDString(sb.UUID)
		label = string(bytes.TrimRightFunc(sb.FilesystemName[:], func(r rune) bool { return r == 0 }))
		totalCapacity = sb.TotalBlocks * uint64(sb.BlockSize)
		freeCapacity = sb.FreeBlocks * uint64(sb.BlockSize)
	} else {
		err = ErrFSNotFound
	}

	return
}

// probe probes FSUUID, total and free capacity.
func probe(path string) (fsuuid, label string, totalCapacity, freeCapacity uint64, err error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()

	doneCh := make(chan struct{})
	go func() {
		var devFile *os.File
		devFile, err = os.OpenFile(path, os.O_RDONLY, os.ModeDevice)
		if err != nil {
			return
		}
		defer devFile.Close()
		// only XFS is the supported filesystem as of now
		fsuuid, label, totalCapacity, freeCapacity, err = readSuperBlock(devFile)
		close(doneCh)
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			klog.InfoS("XFS probe is taking too long; still waiting", "device", path)
		case <-ctx.Done():
			err = errors.Join(ErrCanceled, ctx.Err())
			return
		case <-doneCh:
			return fsuuid, label, totalCapacity, freeCapacity, err
		}
	}
}
