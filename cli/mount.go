// This file is part of MinIO Kubernetes Cloud
// Copyright (c) 2020 MinIO, Inc.
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

package cli

import (
	"os"

	"k8s.io/utils/exec"
	"k8s.io/utils/mount"

	"github.com/minio/direct-csi/pkg/volume"
	"github.com/minio/minio/pkg/disk"
)

// FormatMounter node Mounter - safe format and mount
type FormatMounter struct {
	mount.SafeFormatAndMount
}

func newFormatMounter() *FormatMounter {
	return &FormatMounter{
		mount.SafeFormatAndMount{
			Interface: mount.New(""),
			Exec:      exec.New(),
		},
	}
}

func (m *FormatMounter) PathIsDevice(pathname string) (bool, error) {
	info, err := os.Stat(pathname)
	if err != nil {
		return false, err
	}

	// checks whether the mode is the target mode.
	isSpecificMode := func(mode, targetMode os.FileMode) bool {
		return mode&targetMode == targetMode
	}
	return isSpecificMode(info.Mode(), os.ModeDevice), nil
}

func (m *FormatMounter) MakeDir(pathname string) error {
	// makeDir creates a new directory.
	// If pathname already exists as a directory, no error is returned.
	// If pathname already exists as a file, an error is returned.
	err := os.MkdirAll(pathname, os.FileMode(0755))
	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (m *FormatMounter) GetDiskInfo(mountPath string) (volume.DriveInfo, error) {
	di, err := disk.GetInfo(mountPath)
	if err != nil {
		return volume.DriveInfo{}, err
	}
	drivePath, err := m.GetDeviceName(mountPath)
	if err != nil {
		return volume.DriveInfo{}, err
	}
	return volume.DriveInfo{
		MountPath: mountPath,
		DrivePath: drivePath,
		SysInfo:   di,
	}, nil
}

func (m *FormatMounter) GetDeviceName(mountPath string) (string, error) {
	mntpath, _, err := mount.GetDeviceNameFromMount(m, mountPath)
	return mntpath, err
}
