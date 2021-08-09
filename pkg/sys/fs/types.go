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
	"context"
	"encoding/binary"
)

type Filesystem interface {
	Type() string
	ProbeFS(devicePath string, startOffset int64) (bool, error)
	UUID() (string, error)
	FSBlockSize() uint64
	TotalCapacity() uint64
	FreeCapacity() uint64
	ByteOrder() binary.ByteOrder
}

type Dqblk struct {
	// Definition since Linux 2.4.22
	DqbBHardlimit uint64 // Absolute limit on disk quota blocks alloc
	DqbBSoftlimit uint64 // Preferred limit on disk quota blocks
	DqbCurSpace   uint64 // Current occupied space (in bytes)
	DqbIHardlimit uint64 // Maximum number of allocated inodes
	DqbISoftlimit uint64 // Preferred inode limit
	DqbCurInodes  uint64 // Current number of allocated inodes
	DqbBTime      uint64 // Time limit for excessive disk use
	DqbITime      uint64 // Time limit for excessive files
	DqbValid      uint32 // Bit mask of QIF_* constantss
}

type FSXAttr struct {
	FSXXFlags     uint32
	FSXExtSize    uint32
	FSXNextents   uint32
	FSXProjID     uint32
	FSXCowextSize uint32
	FSXPad        [8]byte
}

type FSQuota struct {
	BlockFile string
	Path      string
	VolumeID  string
}

type Quotaer interface {
	GetVolumeID() string
	GetPath() string
	GetBlockFile() string
	SetQuota(ctx context.Context, limit int64) error
	SetProjectID(projectID uint32) error
	SetProjectQuota(maxBytes uint64, projID uint32) error
	GetQuota() (result *Dqblk, err error)
}
