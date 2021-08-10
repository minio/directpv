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
	// Project Quota
	// -------------------------------
	// subCmdShift = 8
	// subCmdMask  = 0x00ff
	//
	// func qCmd(subCmd, qType int) int {
	//     return subCmd<<subCmdShift | qType&subCmdMask
	// }
	// -------------------------------
	// qGetQuota = 0x800007
	// qSetQuota = 0x800008
	// prjQuota = 2
	//
	getPrjQuotaSubCmd = 0x80000702 // qCmd(qGetQuota, PrjQuota)
	setPrjQuotaSubCmd = 0x80000802 // qCmd(qSetQuota, PrjQuota)

	// Get/Set FS attributes
	fsGetAttr = 0x801c581f // FS_IOC_FSGETXATTR
	fsSetAttr = 0x401c5820 // FS_IOC_FSSETXATTR

	blockSize          = 1024
	flagBLimitsValid   = 1
	flagProjectInherit = 0x00000200
)
