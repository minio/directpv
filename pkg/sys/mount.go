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

// SafeMount mounts device to target using fsType, flags and superBlockFlags.
// Ignores if the target is already mounted to source
func SafeMount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return safeMount(proc1Mountinfo, device, target, fsType, flags, superBlockFlags)
}

// BindMount does bind-mount of source to target.
func BindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return bindMount(proc1Mountinfo, source, target, fsType, recursive, readOnly, superBlockFlags)
}

// SafeUnmount unmounts target with force, detach and expire options.
// Ignores is the target is already umounted
func SafeUnmount(target string, force, detach, expire bool) error {
	return safeUnmount(proc1Mountinfo, target, force, detach, expire)
}

// Unmount unmounts target with force, detach and expire options.
func Unmount(target string, force, detach, expire bool) error {
	return unmount(target, force, detach, expire)
}
