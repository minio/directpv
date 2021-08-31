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

const (
	// For reference :-
	// - https://man7.org/linux/man-pages/man2/quotactl.2.html
	// - https://github.com/torvalds/linux/blob/master/include/uapi/linux/dqblk_xfs.h
	prjQuotaType = 2
	subCmdShift  = 8
	subCmdMask   = 0x00ff

	// Get/Set Quota
	setQuotaLimit    = 0x5804                                                        // Q_XSETQLIM
	prjSetQuotaLimit = uintptr(setQuotaLimit<<subCmdShift | prjQuotaType&subCmdMask) // qCmd(Q_XSETQLIM, PrjQuota)

	getQuota    = 0x5803                                                   // Q_XGETQUOTA
	prjGetQuota = uintptr(getQuota<<subCmdShift | prjQuotaType&subCmdMask) // qCmd(Q_XGETQUOTA, PrjQuota)

	fsDiskQuotaVersion  = 1
	xfsProjectQuotaFlag = 2
	fieldMaskBHard      = 8   // d_blk_hardlimit Field specifier
	fieldMaskBSoft      = 4   // d_blk_softlimit Field specifier
	blockSize           = 512 // All the blk units are in BBs (Basic Blocks) of 512 bytes

	// Get/Set FS attributes
	fsGetAttr          = 0x801c581f // FS_IOC_FSGETXATTR
	fsSetAttr          = 0x401c5820 // FS_IOC_FSSETXATTR
	flagProjectInherit = 0x00000200
)
