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
	"encoding/binary"
	"math"
	"os"

	e "github.com/minio/direct-csi/pkg/sys/ext4"
	x "github.com/minio/direct-csi/pkg/sys/xfs"
)

func (b *BlockDevice) probeFS(offsetBlocks uint64) (*FSInfo, error) {
	ext4FSInfo, err := b.probeFSEXT4(offsetBlocks)
	if err != nil {
		if err != e.ErrNotEXT4 {
			return nil, err
		}
	}
	if ext4FSInfo != nil {
		return ext4FSInfo, nil
	}

	XFSInfo, err := b.probeFSXFS(offsetBlocks)
	if err != nil {
		if err != x.ErrNotXFS {
			return nil, err
		}
	}
	if XFSInfo != nil {
		return XFSInfo, nil
	}

	return nil, ErrNoFS
}

func (b *BlockDevice) probeFSEXT4(offsetBlocks uint64) (*FSInfo, error) {
	devPath := b.DirectCSIDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	_, err = devFile.Seek(int64(b.LogicalBlockSize*offsetBlocks), os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	ext4 := &e.EXT4SuperBlock{}
	err = binary.Read(devFile, binary.LittleEndian, ext4)
	if err != nil {
		return nil, err
	}
	if !ext4.Is() {
		return nil, e.ErrNotEXT4
	}

	fsBlockSize := uint64(math.Pow(2, float64(10+ext4.LogBlockSize)))
	fsInfo := &FSInfo{
		FSType:        e.FSTypeEXT4,
		FSBlockSize:   fsBlockSize,
		TotalCapacity: uint64(ext4.NumBlocks) * uint64(fsBlockSize),
		FreeCapacity:  uint64(ext4.FreeBlocks) * uint64(fsBlockSize),
		Mounts:        []MountInfo{},
	}

	return fsInfo, nil
}

func (b *BlockDevice) probeFSXFS(offsetBlocks uint64) (*FSInfo, error) {
	devPath := b.DirectCSIDrivePath()
	devFile, err := os.OpenFile(devPath, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	_, err = devFile.Seek(int64(b.LogicalBlockSize*offsetBlocks), os.SEEK_CUR)
	if err != nil {
		return nil, err
	}

	xfs := &x.XFSSuperBlock{}
	err = binary.Read(devFile, binary.BigEndian, xfs)
	if err != nil {
		return nil, err
	}

	if !xfs.Is() {
		return nil, x.ErrNotXFS
	}

	fsInfo := &FSInfo{
		FSType:        x.FSTypeXFS,
		FSBlockSize:   uint64(xfs.BlockSize),
		TotalCapacity: uint64(xfs.TotalBlocks) * uint64(xfs.BlockSize),
		FreeCapacity:  uint64(xfs.FreeBlocks) * uint64(xfs.BlockSize),
		Mounts:        []MountInfo{},
	}

	return fsInfo, nil
}
