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
	fs "github.com/minio/direct-csi/pkg/sys/fs"
	ext4 "github.com/minio/direct-csi/pkg/sys/fs/ext4"
	fat32 "github.com/minio/direct-csi/pkg/sys/fs/fat32"
	xfs "github.com/minio/direct-csi/pkg/sys/fs/xfs"
)

func (b *BlockDevice) probeSuperBlocks(offsetBlocks uint64) (fs.Filesystem, error) {

	filesystems := []fs.Filesystem{
		xfs.NewXFS(),
		ext4.NewEXT4(),
		fat32.NewFAT32(),
		// Add new filesystems here
	}

	for _, fs := range filesystems {
		is, err := fs.ProbeFS(b.HostDrivePath(), int64(b.LogicalBlockSize*offsetBlocks))
		if err != nil {
			return nil, err
		}
		if is {
			return fs, nil
		}
	}

	return nil, ErrNoFS
}

func (b *BlockDevice) probeFS(offsetBlocks uint64) (*FSInfo, error) {
	fs, err := b.probeSuperBlocks(offsetBlocks)
	if err != nil {
		return nil, err
	}

	uuid, err := fs.UUID()
	if err != nil {
		return nil, err
	}

	fsInfo := &FSInfo{
		UUID:          uuid,
		FSType:        fs.Type(),
		FSBlockSize:   fs.FSBlockSize(),
		TotalCapacity: fs.TotalCapacity(),
		FreeCapacity:  fs.FreeCapacity(),
		Mounts:        []MountInfo{},
	}

	return fsInfo, nil
}
