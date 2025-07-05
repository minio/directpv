//go:build linux

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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/cespare/xxhash/v2"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func parseMount(s string, xxHash uint64) *MountEntry {
	// Refer https://man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html
	// to know about this logic.
	s = strings.TrimSpace(s)
	tokens := strings.Fields(s)
	if len(tokens) < 8 {
		return nil
	}

	mountOptions := make(utils.StringSet)
	for _, option := range strings.Split(tokens[5], ",") {
		mountOptions.Set(option)
	}

	mount := MountEntry{
		xxHash:     xxHash,
		MountID:    tokens[0],
		ParentID:   tokens[1],
		MajorMinor: tokens[2],
		Root:       tokens[3],
		MountPoint: tokens[4],
	}
	var i int
	for i = 6; i < len(tokens) && tokens[i] != "-"; i++ {
		mount.OptionalFields = append(mount.OptionalFields, tokens[i])
	}
	mount.FilesystemType = tokens[i+1]
	mount.MountSource = tokens[i+2]

	for _, option := range strings.Split(tokens[i+3], ",") {
		mountOptions.Set(option)
	}

	mount.MountOptions = mountOptions
	return &mount
}

func parseMountInfo(r io.Reader, info *MountInfo) (*MountInfo, error) {
	if info == nil {
		info = &MountInfo{
			infoMap:        map[uint64]*MountEntry{},
			mountPointMap:  map[string][]uint64{},
			mountSourceMap: map[string][]uint64{},
			majorMinorMap:  map[string][]uint64{},
			rootMap:        map[string][]uint64{},
		}
	}

	infoMap := info.infoMap
	mountPointMap := info.mountPointMap
	mountSourceMap := info.mountSourceMap
	majorMinorMap := info.majorMinorMap
	rootMap := info.rootMap

	reader := bufio.NewReader(r)

	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		xxHash := xxhash.Sum64String(s)
		if _, found := infoMap[xxHash]; found {
			continue
		}

		mount := parseMount(s, xxHash)
		if mount == nil {
			continue
		}

		infoMap[mount.xxHash] = mount
		mountPointMap[mount.MountPoint] = append(mountPointMap[mount.MountPoint], mount.xxHash)
		mountSourceMap[mount.MountSource] = append(mountSourceMap[mount.MountSource], mount.xxHash)
		majorMinorMap[mount.MajorMinor] = append(majorMinorMap[mount.MajorMinor], mount.xxHash)
		rootMap[mount.Root] = append(rootMap[mount.Root], mount.xxHash)
	}

	return info, nil
}

func newMountInfo() (*MountInfo, error) {
	parseSelfInfo := func() (*MountInfo, error) {
		file, err := os.Open("/proc/self/mountinfo")
		if err != nil {
			return nil, err
		}

		defer file.Close()
		return parseMountInfo(file, nil)
	}

	parseRootInfo := func(info *MountInfo) (*MountInfo, error) {
		file, err := os.Open("/proc/1/mountinfo")
		if err != nil {
			switch {
			case errors.Is(err, os.ErrInvalid), errors.Is(err, os.ErrPermission), errors.Is(err, os.ErrExist), errors.Is(err, os.ErrNotExist), errors.Is(err, os.ErrClosed):
			default:
				klog.ErrorS(err, "unable to read /proc/1/mountinfo; open an issue in SUBNET with this log")
			}
			return info, nil
		}

		defer file.Close()
		return parseMountInfo(file, info)
	}

	info, err := parseSelfInfo()
	if err != nil {
		return nil, err
	}

	return parseRootInfo(info)
}

var mountFlagMap = map[string]uintptr{
	"remount":     syscall.MS_REMOUNT,
	"bind":        syscall.MS_BIND,
	"shared":      syscall.MS_SHARED,
	"private":     syscall.MS_PRIVATE,
	"slave":       syscall.MS_SLAVE,
	"unbindable":  syscall.MS_UNBINDABLE,
	"move":        syscall.MS_MOVE,
	"dirsync":     syscall.MS_DIRSYNC,
	"mand":        syscall.MS_MANDLOCK,
	"noatime":     syscall.MS_NOATIME,
	"nodev":       syscall.MS_NODEV,
	"nodiratime":  syscall.MS_NODIRATIME,
	"noexec":      syscall.MS_NOEXEC,
	"nosuid":      syscall.MS_NOSUID,
	"ro":          syscall.MS_RDONLY,
	"relatime":    syscall.MS_RELATIME,
	"recursive":   syscall.MS_REC,
	"silent":      syscall.MS_SILENT,
	"strictatime": syscall.MS_STRICTATIME,
	"sync":        syscall.MS_SYNCHRONOUS,
}

func mount(device, target, fsType string, flags []string, superBlockFlags string) error {
	mountInfo, err := newMountInfo()
	if err != nil {
		return err
	}

	if mountInfo = mountInfo.FilterByMountPoint(target); !mountInfo.IsEmpty() {
		if !mountInfo.FilterByMountSource(device).IsEmpty() {
			klog.V(5).InfoS("device is already mounted on target", "device", device, "target", target, "fsType", fsType, "flags", flags, "superBlockFlags", superBlockFlags)
			return nil
		}

		var devices []string
		for _, mount := range mountInfo.List() {
			devices = append(devices, mount.MountSource)
		}
		return &ErrMountPointAlreadyMounted{MountPoint: target, Devices: devices}
	}

	mountFlags := uintptr(0)
	for _, flag := range flags {
		value, found := mountFlagMap[flag]
		if !found {
			return fmt.Errorf("unknown flag %v", flag)
		}
		mountFlags |= value
	}
	klog.V(5).InfoS("mounting device", "device", device, "target", target, "fsType", fsType, "mountFlags", mountFlags, "superBlockFlags", superBlockFlags)
	return syscall.Mount(device, target, fsType, mountFlags, superBlockFlags)
}

func bindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	flags := mountFlagMap["bind"]
	if recursive {
		flags |= mountFlagMap["recursive"]
	}
	if readOnly {
		flags |= mountFlagMap["ro"]
	}
	klog.V(5).InfoS("bind mounting directory", "source", source, "target", target, "fsType", fsType, "recursive", recursive, "readOnly", readOnly, "superBlockFlags", superBlockFlags)
	return syscall.Mount(source, target, fsType, flags, superBlockFlags)
}

func unmount(target string, force, detach, expire bool) error {
	mountInfo, err := newMountInfo()
	if err != nil {
		return err
	}

	if mountInfo.FilterByMountPoint(target).IsEmpty() {
		klog.V(5).InfoS("target already unmounted", "target", target, "force", force, "detach", detach, "expire", expire)
		return nil
	}

	flags := 0
	if force {
		flags |= syscall.MNT_FORCE
	}
	if detach {
		flags |= syscall.MNT_DETACH
	}
	if expire {
		flags |= syscall.MNT_EXPIRE
	}

	klog.V(5).InfoS("unmounting mount point", "target", target, "force", force, "detach", detach, "expire", expire)
	return syscall.Unmount(target, flags)
}
