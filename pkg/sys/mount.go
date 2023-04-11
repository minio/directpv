// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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
	"fmt"
	"path/filepath"

	"github.com/minio/directpv/pkg/utils"
)

// ErrMountPointAlreadyMounted denotes mount point already mounted error.
type ErrMountPointAlreadyMounted struct {
	MountPoint string
	Devices    []string
}

// Error is error interface compatible method.
func (e *ErrMountPointAlreadyMounted) Error() string {
	return fmt.Sprintf("mount point %v already mounted by %v", e.MountPoint, e.Devices)
}

// GetMounts returns mount-point to devices and devices to mount-point maps.
func GetMounts(includeMajorMinorMap bool) (mountPointMap, deviceMap, majorMinorMap, rootMountPointMap map[string]utils.StringSet, err error) {
	return getMounts(includeMajorMinorMap)
}

// Mount mounts device to target using fsType, flags and superBlockFlags.
func Mount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return mount(device, target, fsType, flags, superBlockFlags)
}

// BindMount does bind-mount of source to target.
func BindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return bindMount(source, target, fsType, recursive, readOnly, superBlockFlags)
}

// Unmount unmounts target with force, detach and expire options.
func Unmount(target string, force, detach, expire bool) error {
	return unmount(target, force, detach, expire)
}

// GetDeviceByFSUUID get device name by it's FSUUID.
func GetDeviceByFSUUID(fsuuid string) (device string, err error) {
	if device, err = filepath.EvalSymlinks("/dev/disk/by-uuid/" + fsuuid); err == nil {
		device = filepath.ToSlash(device)
	}
	return
}
