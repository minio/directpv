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

// MountEntry contains single mount information from /proc/self/mountinfo
type MountEntry struct {
	xxHash         uint64
	MountID        string
	ParentID       string
	MajorMinor     string
	Root           string
	MountPoint     string
	MountOptions   utils.StringSet
	OptionalFields []string
	FilesystemType string
	MountSource    string
}

func (m MountEntry) String() string {
	return fmt.Sprintf("%#+v", m)
}

// MountInfo contains multiple mount information from /proc/self/mountinfo
type MountInfo struct {
	infoMap        map[uint64]*MountEntry
	mountPointMap  map[string][]uint64
	mountSourceMap map[string][]uint64
	majorMinorMap  map[string][]uint64
	rootMap        map[string][]uint64
}

// NewMountInfo creates mount information from /proc/self/mountinfo
func NewMountInfo() (*MountInfo, error) {
	return newMountInfo()
}

// IsEmpty checks whether this mount information is empty or not
func (info *MountInfo) IsEmpty() bool {
	return len(info.infoMap) == 0
}

// Length returns number of mount information available
func (info *MountInfo) Length() int {
	return len(info.infoMap)
}

// List returns list of mount entries available
func (info *MountInfo) List() (mounts []*MountEntry) {
	for _, mount := range info.infoMap {
		mounts = append(mounts, mount)
	}
	return
}

func (info *MountInfo) filter(xxHashes []uint64) *MountInfo {
	infoMap := map[uint64]*MountEntry{}
	mountPointMap := map[string][]uint64{}
	mountSourceMap := map[string][]uint64{}
	majorMinorMap := map[string][]uint64{}
	rootMap := map[string][]uint64{}

	for _, xxHash := range xxHashes {
		if mount, found := info.infoMap[xxHash]; found {
			infoMap[xxHash] = mount
			mountPointMap[mount.MountPoint] = append(mountPointMap[mount.MountPoint], mount.xxHash)
			mountSourceMap[mount.MountSource] = append(mountSourceMap[mount.MountSource], xxHash)
			majorMinorMap[mount.MajorMinor] = append(majorMinorMap[mount.MajorMinor], mount.xxHash)
			rootMap[mount.Root] = append(rootMap[mount.Root], mount.xxHash)
		}
	}

	return &MountInfo{
		infoMap:        infoMap,
		mountPointMap:  mountPointMap,
		mountSourceMap: mountSourceMap,
		majorMinorMap:  majorMinorMap,
		rootMap:        rootMap,
	}
}

// FilterByMountPoint returns mount information filtered by mount point
func (info *MountInfo) FilterByMountPoint(value string) *MountInfo {
	return info.filter(info.mountPointMap[value])
}

// FilterByMountSource returns mount information filtered by mount source
func (info *MountInfo) FilterByMountSource(value string) *MountInfo {
	return info.filter(info.mountSourceMap[value])
}

// FilterByMajorMinor returns mount information filtered by major:minor
func (info *MountInfo) FilterByMajorMinor(value string) *MountInfo {
	return info.filter(info.majorMinorMap[value])
}

// FilterByRoot returns mount information filtered by root mount point
func (info *MountInfo) FilterByRoot(value string) *MountInfo {
	return info.filter(info.rootMap[value])
}

// ErrMountPointAlreadyMounted denotes mount point already mounted error.
type ErrMountPointAlreadyMounted struct {
	MountPoint string
	Devices    []string
}

// Error is error interface compatible method.
func (e *ErrMountPointAlreadyMounted) Error() string {
	return fmt.Sprintf("mount point %v already mounted by %v", e.MountPoint, e.Devices)
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
