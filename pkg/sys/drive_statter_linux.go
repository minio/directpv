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
	"syscall"
)

func getFreeCapacityFromStatfs(path string) (freeCapacity int64, err error) {
	stat := &syscall.Statfs_t{}
	err = syscall.Statfs(path, stat)
	if err != nil {
		return
	}
	freeCapacity = int64(stat.Frsize) * int64(stat.Bavail)
	return
}

// DriveStatter denotes function to get free capacity of a drive.
type DriveStatter interface {
	GetFreeCapacityFromStatfs(path string) (freeCapacity int64, err error)
}

// DefaultDriveStatter is a default interface to get free capacity of a drive.
type DefaultDriveStatter struct{}

// GetFreeCapacityFromStatfs gets free capacity of a drive.
func (c *DefaultDriveStatter) GetFreeCapacityFromStatfs(path string) (int64, error) {
	return getFreeCapacityFromStatfs(path)
}
