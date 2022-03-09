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

package mount

import (
	"os"

	"k8s.io/klog/v2"
)

const (
	// MountOptPrjQuota option for project quota
	MountOptPrjQuota = "prjquota"
)

// Mount mounts device to target using fsType, flags and superBlockFlags.
func Mount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return mount(device, target, fsType, flags, superBlockFlags)
}

// Unmount unmounts target with force, detach and expire options.
func Unmount(target string, force, detach, expire bool) error {
	return unmount(target, force, detach, expire)
}

// SafeMount mounts device only if target is not a mount point.
func SafeMount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return safeMount(device, target, fsType, flags, superBlockFlags)
}

// SafeBindMount does bind-mount of source to target only if target is not a mount point.
func SafeBindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return safeBindMount(source, target, fsType, recursive, readOnly, superBlockFlags)
}

// SafeUnmount unmount if target is a mount point.
func SafeUnmount(target string, force, detach, expire bool) error {
	return safeUnmount(target, force, detach, expire)
}

// UnmountDevice unmounts all mounts of device.
func UnmountDevice(device string) error {
	return unmountDevice(device)
}

// MountXFSDevice mounts device having XFS filesystem into target.
func MountXFSDevice(device, target string, flags []string) error {
	if err := os.MkdirAll(target, 0777); err != nil {
		return err
	}

	klog.V(3).InfoS("mounting device", "device", device, "target", target)
	return SafeMount(device, target, "xfs", flags, MountOptPrjQuota)
}
