// +build !linux

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

type DriveMounter interface {
	MountDrive(source, target string, mountOpts []string) error
	UnmountDrive(source string) error
}

type DefaultDriveMounter struct{}

func (c *DefaultDriveMounter) MountDrive(source, target string, mountOpts []string) error {
	return nil
}

func (c *DefaultDriveMounter) UnmountDrive(source string) error {
	return nil
}
