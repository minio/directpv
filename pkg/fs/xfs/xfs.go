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
	"fmt"
	"io"

	fserrors "github.com/minio/directpv/pkg/fs/errors"
)

// UUID2String converts UUID to string.
func UUID2String(uuid [16]byte) string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%x-%x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		uuid[8:10],
		uuid[10:],
	)
}

// SuperBlock denotes XFS superblock.
type SuperBlock struct {
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

// ID returns filesystem UUID.
func (sb *SuperBlock) ID() string {
	return UUID2String(sb.UUID)
}

// Type returns "xfs".
func (sb *SuperBlock) Type() string {
	return "xfs"
}

// TotalCapacity returns total capacity of filesystem.
func (sb *SuperBlock) TotalCapacity() uint64 {
	return sb.TotalBlocks * uint64(sb.BlockSize)
}

// FreeCapacity returns free capacity of filesystem.
func (sb *SuperBlock) FreeCapacity() uint64 {
	return sb.FreeBlocks * uint64(sb.BlockSize)
}

// Probe tries to probe XFS superblock.
func Probe(reader io.Reader) (*SuperBlock, error) {
	var superBlock SuperBlock
	if err := binary.Read(reader, binary.BigEndian, &superBlock); err != nil {
		return nil, err
	}

	if superBlock.MagicNumber != 0x58465342 {
		return nil, fserrors.ErrFSNotFound
	}

	return &superBlock, nil
}
