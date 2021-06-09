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

package fs

import (
	"encoding/binary"
	"errors"

	"github.com/minio/direct-csi/pkg/sys/fs/ext4"
	"github.com/minio/direct-csi/pkg/sys/fs/fat32"
	"github.com/minio/direct-csi/pkg/sys/fs/swap"
	"github.com/minio/direct-csi/pkg/sys/fs/xfs"
)

var ErrFSNotFound = errors.New("file system not found")

type Filesystem interface {
	Type() string
	ProbeFS(devicePath string, startOffset int64) (bool, error)
	UUID() (string, error)
	FSBlockSize() uint64
	TotalCapacity() uint64
	FreeCapacity() uint64
	ByteOrder() binary.ByteOrder
}

func Probe(device string) (Filesystem, error) {
	filesystems := []Filesystem{
		xfs.NewXFS(),
		ext4.NewEXT4(),
		fat32.NewFAT32(),
		swap.NewSwap(),
		// Add new filesystems here
	}

	for _, fs := range filesystems {
		found, err := fs.ProbeFS(device, 0)
		if err != nil {
			return nil, err
		}
		if found {
			return fs, nil
		}
	}

	return nil, ErrFSNotFound
}

func GetCapacity(device, filesystem string) (totalCapacity, freeCapacity uint64, err error) {
	var fs Filesystem
	switch filesystem {
	case "xfs":
		fs = xfs.NewXFS()
	case "ext4":
		fs = ext4.NewEXT4()
	case "vfat":
		fs = fat32.NewFAT32()
	case "swap":
		return 0, 0, nil
	default:
		return 0, 0, ErrFSNotFound
	}

	found, err := fs.ProbeFS(device, 0)
	if err != nil {
		return 0, 0, err
	}
	if !found {
		return 0, 0, ErrFSNotFound
	}

	return fs.TotalCapacity(), fs.FreeCapacity(), nil
}
