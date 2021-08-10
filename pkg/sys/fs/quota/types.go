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

package quota

import (
	"context"
)

type dqblk struct {
	// Definition since Linux 2.4.22
	dqbBHardlimit uint64 // Absolute limit on disk quota blocks alloc
	dqbBSoftlimit uint64 // Preferred limit on disk quota blocks
	dqbCurSpace   uint64 // Current occupied space (in bytes)
	dqbIHardlimit uint64 // Maximum number of allocated inodes
	dqbISoftlimit uint64 // Preferred inode limit
	dqbCurInodes  uint64 // Current number of allocated inodes
	dqbBTime      uint64 // Time limit for excessive disk use
	dqbITime      uint64 // Time limit for excessive files
	dqbValid      uint32 // Bit mask of QIF_* constantss
}

type fsXAttr struct {
	fsXXFlags     uint32
	fsXExtSize    uint32
	fsXNextents   uint32
	fsXProjID     uint32
	fsXCowextSize uint32
	fsXPad        [8]byte
}

type FSQuota struct {
	HardLimit    int64
	SoftLimit    int64
	CurrentSpace int64
}

type Quotaer interface {
	SetQuota(ctx context.Context, path, volumeID, blockFile string, quota FSQuota) error
	GetQuota(blockFile, volumeID string) (FSQuota, error)
}
