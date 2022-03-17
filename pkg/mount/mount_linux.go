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

package mount

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

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
	mountFlags := uintptr(0)
	for _, flag := range flags {
		value, found := mountFlagMap[flag]
		if !found {
			return fmt.Errorf("unknown flag %v", flag)
		}
		mountFlags |= value
	}
	return syscall.Mount(device, target, fsType, mountFlags, superBlockFlags)
}

func unmount(target string, force, detach, expire bool) error {
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

func IsMounted(target string) (bool, error) {
	mountInfos, err := Probe()
	if err != nil {
		return false, err
	}

	for _, mounts := range mountInfos {
		for _, mount := range mounts {
			if mount.MountPoint == target {
				return true, nil
			}
		}
	}

	return false, nil
}

func safeMount(device, target, fsType string, flags []string, superBlockFlags string) error {
	mounted, err := IsMounted(target)
	if err != nil {
		return err
	}
	if mounted {
		klog.V(5).InfoS("target already mounted", "device", device, "target", target, "fsType", fsType, "flags", flags, "superBlockFlags", superBlockFlags)
		return nil
	}
	return mount(device, target, fsType, flags, superBlockFlags)
}

func safeBindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	mounted, err := IsMounted(target)
	if err != nil {
		return err
	}
	if mounted {
		klog.V(5).InfoS("target already mounted", "source", source, "target", target, "fsType", fsType, "recursive", recursive, "superBlockFlags", superBlockFlags)
		return nil
	}

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

func safeUnmount(target string, force, detach, expire bool) error {
	mounted, err := IsMounted(target)
	if err != nil {
		return err
	}
	if !mounted {
		klog.V(5).InfoS("target already unmounted", "target", target, "force", force, "detach", detach, "expire", expire)
		return nil
	}
	return unmount(target, force, detach, expire)
}

func getDeviceMajorMinor(device string) (major, minor uint32, err error) {
	stat := syscall.Stat_t{}
	if err = syscall.Stat(device, &stat); err == nil {
		major, minor = uint32(unix.Major(stat.Rdev)), uint32(unix.Minor(stat.Rdev))
	}
	return
}

func unmountDevice(device string) error {
	major, minor, err := getDeviceMajorMinor(device)
	if err != nil {
		return err
	}

	mountInfos, err := Probe()
	if err != nil {
		return err
	}

	if mounts, found := mountInfos[fmt.Sprintf("%v:%v", major, minor)]; found {
		for _, mount := range mounts {
			if err := unmount(mount.MountPoint, true, true, false); err != nil {
				return err
			}
		}
	}

	return nil
}
