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

package xfs

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/google/uuid"
	"os"
)

type XFS struct {
	SuperBlock *XFSSuperBlock
}

func NewXFS() *XFS {
	return &XFS{}
}

type XFSSuperBlock struct {
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

type XFSVolumeStats struct {
	AvailableBytes int64
	TotalBytes     int64
	UsedBytes      int64
}

func (x XFSSuperBlock) Is() bool {
	return x.MagicNumber == XFSMagicNum
}

func (xfs *XFS) Type() string {
	return FSTypeXFS
}

func (xfs *XFS) FSBlockSize() uint64 {
	return uint64(xfs.SuperBlock.BlockSize)
}

func (xfs *XFS) TotalCapacity() uint64 {
	return uint64(xfs.SuperBlock.TotalBlocks) * uint64(xfs.SuperBlock.BlockSize)
}

func (xfs *XFS) FreeCapacity() uint64 {
	return uint64(xfs.SuperBlock.FreeBlocks) * uint64(xfs.SuperBlock.BlockSize)
}

func (xfs *XFS) UUID() (string, error) {
	uid, err := uuid.Parse(hex.EncodeToString(xfs.SuperBlock.UUID[:]))
	if err != nil {
		return "", err
	}
	return uid.String(), nil
}

func (xfs *XFS) ByteOrder() binary.ByteOrder {
	return binary.BigEndian
}

func (xfs *XFS) ProbeFS(devicePath string, startOffset int64) (bool, error) {
	devFile, err := os.OpenFile(devicePath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return false, err
	}
	defer devFile.Close()

	if _, err = devFile.Seek(startOffset, os.SEEK_SET); err != nil {
		return false, err
	}

	xfsSuperBlock := &XFSSuperBlock{}
	if err := binary.Read(devFile, xfs.ByteOrder(), xfsSuperBlock); err != nil {
		return false, err
	}
	xfs.SuperBlock = xfsSuperBlock

	return xfs.SuperBlock.Is(), nil
}
