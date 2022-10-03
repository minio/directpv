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

const proc1Mountinfo = "/proc/1/mountinfo"

func GetMounts() (mountPointMap, deviceMap map[string][]string, err error) {
	return getMounts(proc1Mountinfo)
}

// Mount mounts device to target using fsType, flags and superBlockFlags.
func Mount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return mount(proc1Mountinfo, device, target, fsType, flags, superBlockFlags)
}

// BindMount does bind-mount of source to target.
func BindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return bindMount(proc1Mountinfo, source, target, fsType, recursive, readOnly, superBlockFlags)
}

// Unmount unmounts target with force, detach and expire options.
func Unmount(target string, force, detach, expire bool) error {
	return unmount(proc1Mountinfo, target, force, detach, expire)
}

// UnmountDriveMounts umounts the device with force, detach and expire options
func UnmountDriveMounts(devicePath string, force, detach, expire bool) error {
	return unmountDriveMounts(proc1Mountinfo, devicePath, force, detach, expire)
}
