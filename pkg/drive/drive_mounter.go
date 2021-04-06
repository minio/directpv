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

package drive

import (
	"github.com/minio/direct-csi/pkg/utils"
)

type DriveMounter interface {
	MountDrive(source, target string, mountOpts []string) error
	UnmountDrive(source string) error
}

type driveMounter struct{}

func (c *driveMounter) MountDrive(source, target string, mountOpts []string) error {
	return mountDrive(source, target, mountOpts)
}

func (c *driveMounter) UnmountDrive(source string) error {
	return unmountDrive(source)
}

type fakeDriveMounter struct{}

func (c *fakeDriveMounter) MountDrive(source, target string, mountOpts []string) error {
	return nil
}

func (c *fakeDriveMounter) UnmountDrive(source string) error {
	return nil
}

func GetDriveMounter() DriveMounter {
	if utils.GetFake() {
		return &fakeDriveMounter{}
	}
	return &driveMounter{}
}
