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
	"os"

	"github.com/golang/glog"
)

const (
	quotaOption = "prjquota"
)

// mountDrive - Idempotent function to mount a DirectCSIDrive
func mountDrive(source, target string, mountOpts []string) error {
	// Since pods will be consuming this target, be permissive
	if err := os.MkdirAll(target, 0777); err != nil {
		return err
	}

	glog.V(3).Infof("mounting drive %s at %s", source, target)
	return SafeMount(source, target, string(FSTypeXFS), func(opts []string) []MountOption {
		newOpts := []MountOption{}
		for _, opt := range opts {
			newOpts = append(newOpts, MountOption(opt))
		}
		return newOpts
	}(mountOpts), []string{
		quotaOption,
	})

	return nil
}

// unmountDrive - Idempotent function to unmount a DirectCSIDrive
func unmountDrive(drivePath string) error {
	glog.V(3).Infof("unmounting drive %s", drivePath)
	if err := SafeUnmountAll(drivePath, []UnmountOption{
		UnmountOptionDetach,
		UnmountOptionForce,
	}); err != nil {
		return err
	}

	return nil
}

type DriveMounter interface {
	MountDrive(source, target string, mountOpts []string) error
	UnmountDrive(source string) error
}

type DefaultDriveMounter struct{}

func (c *DefaultDriveMounter) MountDrive(source, target string, mountOpts []string) error {
	return mountDrive(source, target, mountOpts)
}

func (c *DefaultDriveMounter) UnmountDrive(source string) error {
	return unmountDrive(source)
}
