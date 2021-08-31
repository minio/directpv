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

// Ref: https://man7.org/linux/man-pages/man2/quotactl.2.html
type fsDiskQuota struct {
	version         int8    // Version of this structure
	flags           int8    // XFS_{USER,PROJ,GROUP}_QUOTA
	fieldmask       uint16  // Field specifier
	id              uint32  // User, project, or group ID
	hardLimitBlocks uint64  // Absolute limit on disk blocks
	softLimitBlocks uint64  // Preferred limit on disk blocks
	hardLimitInodes uint64  // Maximum allocated inodes
	softLimitInodes uint64  // Preferred inode limit
	blocksCount     uint64  // disk blocks owned by the project/user/group
	inodesCount     uint64  // inodes owned by the project/user/group
	inodeTimer      int32   // Zero if within inode limits, If not, we refuse service */
	blocksTimer     int32   // Similar to above; for disk blocks
	inodeWarnings   uint16  // warnings issued with respect to number of inodes
	blockWarnings   uint16  // warnings issued with respect to disk blocks
	padding2        int32   // Padding - for future use
	rtbHardLimit    uint64  // Absolute limit on realtime (RT) disk blocks
	rtbSoftLimit    uint64  // Preferred limit on RT disk blocks
	rtbCount        uint64  // realtime blocks owned
	rtbTimer        int32   // Similar to above; for RT disk blocks
	rtbWarnings     uint16  // warnings issued with respect to RT disk blocks
	padding3        int16   // Padding - for future use
	padding4        [8]byte // Yet more padding
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
