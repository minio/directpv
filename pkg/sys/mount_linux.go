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
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

func parseProc1Mountinfo(r io.Reader) (mountPointMap, deviceMap, majorMinorMap, rootMountPointMap map[string]utils.StringSet, err error) {
	reader := bufio.NewReader(r)

	mountPointMap = make(map[string]utils.StringSet)
	deviceMap = make(map[string]utils.StringSet)
	majorMinorMap = make(map[string]utils.StringSet)
	rootMountPointMap = make(map[string]utils.StringSet)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, nil, nil, err
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

		majorMinor := tokens[2]
		root := tokens[3]
		mountPoint := tokens[4]
		device := tokens[i+1]

		if _, found := mountPointMap[mountPoint]; !found {
			mountPointMap[mountPoint] = make(utils.StringSet)
		}
		mountPointMap[mountPoint].Set(device)

		if _, found := deviceMap[device]; !found {
			deviceMap[device] = make(utils.StringSet)
		}
		deviceMap[device].Set(mountPoint)

		if _, found := majorMinorMap[majorMinor]; !found {
			majorMinorMap[majorMinor] = make(utils.StringSet)
		}
		majorMinorMap[majorMinor].Set(device)

		if _, found := rootMountPointMap[root]; !found {
			rootMountPointMap[root] = make(utils.StringSet)
		}
		rootMountPointMap[root].Set(mountPoint)
	}

	return
}

func getMajorMinor(device string) (majorMinor string, err error) {
	stat := syscall.Stat_t{}
	if err = syscall.Stat(device, &stat); err == nil {
		majorMinor = fmt.Sprintf("%v:%v", unix.Major(stat.Rdev), unix.Minor(stat.Rdev))
	}
	return
}

func parseProcMounts(r io.Reader, includeMajorMinorMap bool) (mountPointMap, deviceMap, majorMinorMap, rootMountPointMap map[string]utils.StringSet, err error) {
	reader := bufio.NewReader(r)

	mountPointMap = make(map[string]utils.StringSet)
	deviceMap = make(map[string]utils.StringSet)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, nil, nil, err
		}

		// Refer /proc/mounts section in https://man7.org/linux/man-pages/man5/proc.5.html
		// to know about this logic.
		tokens := strings.Fields(strings.TrimSpace(s))
		if len(tokens) < 2 {
			continue
		}

		mountPoint := tokens[1]
		device := tokens[0]

		if _, found := mountPointMap[mountPoint]; !found {
			mountPointMap[mountPoint] = make(utils.StringSet)
		}
		mountPointMap[mountPoint].Set(device)

		if _, found := deviceMap[device]; !found {
			deviceMap[device] = make(utils.StringSet)
		}
		deviceMap[device].Set(mountPoint)
	}

	if !includeMajorMinorMap {
		return
	}

	majorMinorMap = make(map[string]utils.StringSet)
	for device := range deviceMap {
		// Ignore pseudo devices.
		if !strings.HasPrefix(device, "/") {
			continue
		}

		majorMinor, err := getMajorMinor(device)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		if _, found := majorMinorMap[device]; !found {
			majorMinorMap[majorMinor] = make(utils.StringSet)
		}
		majorMinorMap[majorMinor].Set(device)
	}

	return
}

func getMounts(includeMajorMinorMap bool) (mountPointMap, deviceMap, majorMinorMap, rootMountPointMap map[string]utils.StringSet, err error) {
	file, err := os.Open("/proc/1/mountinfo")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, nil, err
		}
	} else {
		defer file.Close()
		return parseProc1Mountinfo(file)
	}

	if file, err = os.Open("/proc/mounts"); err != nil {
		return nil, nil, nil, nil, err
	}

	defer file.Close()
	return parseProcMounts(file, includeMajorMinorMap)
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
	mountPointMap, _, _, _, err := getMounts(false)
	if err != nil {
		return err
	}

	if devices, found := mountPointMap[target]; found {
		if devices.Exist(device) {
			klog.V(5).InfoS("device is already mounted on target", "device", device, "target", target, "fsType", fsType, "flags", flags, "superBlockFlags", superBlockFlags)
			return nil
		}

		return &ErrMountPointAlreadyMounted{MountPoint: target, Devices: devices.ToSlice()}
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
	mountPointMap, _, _, _, err := getMounts(false)
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
