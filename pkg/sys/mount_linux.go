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

	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func parseProc1Mountinfo(r io.Reader) (mountPointMap, deviceMap map[string][]string, err error) {
	reader := bufio.NewReader(r)

	mountPointMap = map[string][]string{}
	deviceMap = map[string][]string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, err
		}

		// Refer /proc/[pid]/mountinfo section in https://man7.org/linux/man-pages/man5/proc.5.html
		// to know about this logic.
		tokens := strings.Fields(strings.TrimSpace(s))
		if len(tokens) < 8 {
			continue
		}

		// Skip mount tags.
		var i int
		for i = 6; i < len(tokens); i++ {
			if tokens[i] == "-" {
				i++
				break
			}
		}

		mountPointMap[tokens[4]] = append(mountPointMap[tokens[4]], tokens[i+1])
		deviceMap[tokens[i+1]] = append(mountPointMap[tokens[i+1]], tokens[4])
	}

	return
}

func parseProcMounts(r io.Reader) (mountPointMap, deviceMap map[string][]string, err error) {
	reader := bufio.NewReader(r)

	mountPointMap = map[string][]string{}
	deviceMap = map[string][]string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, err
		}

		// Refer /proc/mounts section in https://man7.org/linux/man-pages/man5/proc.5.html
		// to know about this logic.
		tokens := strings.Fields(strings.TrimSpace(s))
		if len(tokens) < 2 {
			continue
		}

		mountPointMap[tokens[1]] = append(mountPointMap[tokens[1]], tokens[0])
		deviceMap[tokens[0]] = append(mountPointMap[tokens[0]], tokens[1])
	}

	return
}

func getMounts() (mountPointMap, deviceMap map[string][]string, err error) {
	file, err := os.Open("/proc/1/mountinfo")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		defer file.Close()
		return parseProc1Mountinfo(file)
	}

	if file, err = os.Open("/proc/mounts"); err != nil {
		return nil, nil, err
	}

	defer file.Close()
	return parseProcMounts(file)
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
	mountPointMap, _, err := getMounts()
	if err != nil {
		return err
	}

	if devices, found := mountPointMap[target]; found {
		if utils.Contains(devices, device) {
			klog.V(5).InfoS("device is already mounted on target", "device", device, "target", target, "fsType", fsType, "flags", flags, "superBlockFlags", superBlockFlags)
			return nil
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
	mountPointMap, _, err := getMounts()
	if err != nil {
		return err
	}

	if devices, found := mountPointMap[target]; found {
		if utils.Contains(devices, source) {
			klog.V(5).InfoS("source is already mounted on target", "source", source, "target", target, "fsType", fsType, "recursive", recursive, "readOnly", readOnly, "superBlockFlags", superBlockFlags)
			return nil
		}

		return &ErrMountPointAlreadyMounted{MountPoint: target, Devices: devices}
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

func unmount(target string, force, detach, expire bool) error {
	mountPointMap, _, err := getMounts()
	if err != nil {
		return err
	}

	if _, found := mountPointMap[target]; !found {
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
