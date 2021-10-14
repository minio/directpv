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

	"k8s.io/klog/v2"
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

	klog.V(3).Infof("mounting drive %s at %s", source, target)
	return safeMount(source, target, string(FSTypeXFS), func(opts []string) []MountOption {
		newOpts := []MountOption{}
		for _, opt := range opts {
			newOpts = append(newOpts, MountOption(opt))
		}
		return newOpts
	}(mountOpts), []string{
		quotaOption,
	})
}

// unmountDrive - Idempotent function to unmount a DirectCSIDrive
func unmountDrive(path string) error {
	klog.V(3).Infof("unmounting drive %s", path)
	if err := safeUnmountAll(path, []UnmountOption{
		UnmountOptionDetach,
		UnmountOptionForce,
	}); err != nil {
		return err
	}

	return nil
}

// DriveMounter is mount/unmount drive interface.
type DriveMounter interface {
	MountDrive(source, target string, mountOpts []string) error
	UnmountDrive(path string) error
}

// DefaultDriveMounter is a default mount/unmount drive interface.
type DefaultDriveMounter struct{}

// MountDrive mounts a drive into given mountpoint.
func (c *DefaultDriveMounter) MountDrive(source, target string, mountOpts []string) error {
	return mountDrive(source, target, mountOpts)
}

// UnmountDrive unmounts given mountpoint.
func (c *DefaultDriveMounter) UnmountDrive(path string) error {
	return unmountDrive(path)
}
