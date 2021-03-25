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

package ext4

type EXT4SuperBlock struct {
	_                         [1024]byte
	NumInodes                 uint32
	NumBlocks                 uint32
	ReservedBlocks            uint32
	FreeBlocks                uint32
	FreeInodes                uint32
	FirstDataBlock            uint32
	LogBlockSize              uint32
	LogClusterSize            uint32
	BlocksPerGroup            uint32
	ObsoleteFragmentsPerGroup uint32
	InodesPerGroup            uint32
	MountTime                 uint32
	WriteTime                 uint32
	MountCount                uint16
	MaxMountCount             uint16
	MagicNum                  uint16
	FilesystemState           uint16
	Errors                    uint16
	MinorRevLevel             uint16
	LastCheckTime             uint32
	CheckInterval             uint32
	CreatorOS                 uint32
	RevLevel                  uint32
	DefaultReserveUID         uint16
	DefaultReserveGID         uint16
}

func (e EXT4SuperBlock) Is() bool {
	return e.MagicNum == EXT4MagicNum
}
