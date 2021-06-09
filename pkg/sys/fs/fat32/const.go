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

package fat32

import (
	"errors"
)

const (
	FSTypeFAT32 = "fat32"
	// Magic is the VFAT magic signature.
	FAT32Magic = "FAT32"
	// FAT32FSInfoSectorSignatureStart is the 4 bytes that signify the beginning of a FAT32 FS Information Sector
	FAT32FSInfoSectorSignatureStart uint32 = 0x41615252
	// FAT32FSInfoSectorSignatureMid is the 4 bytes that signify the middle bytes 484-487 of a FAT32 FS Information Sector
	FAT32FSInfoSectorSignatureMid uint32 = 0x61417272
	// FAT32FSInfoSectorSignatureEnd is the 4 bytes that signify the end of a FAT32 FS Information Sector
	FAT32FSInfoSectorSignatureEnd uint32 = 0xAA550000
)

var ErrNotFAT32 = errors.New("Not a fat32 partition")
