//go:build linux

// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"strings"
	"syscall"

	"k8s.io/klog/v2"
)

func SafeMount(source, target, fsType string, mountOpts []MountOption, superblockOpts []string) error {
	mountInfos, err := ProbeMounts()
	if err != nil {
		return err
	}

	major, minor, err := GetMajorMinor(source)
	if err != nil {
		return err
	}

	if mounts, found := mountInfos[fmt.Sprintf("%v:%v", major, minor)]; found {
		for _, mount := range mounts {
			if mount.MountPoint == target {
				klog.V(3).Infof("drive already mounted: %s", target)
				return nil
			}
		}
	}

	return Mount(source, target, fsType, mountOpts, superblockOpts)
}

func Mount(source, target, fsType string, mountOpts []MountOption, superblockOpts []string) error {
	verifyRemount := func(mountOpts []MountOption) error {
		remount := false
		for _, opt := range mountOpts {
			if opt == MountOptionMSRemount {
				remount = true
			}
		}
		if !remount {
			return nil
		}
		for _, opt := range mountOpts {
			switch opt {
			case MountOptionMSMandLock:
			case MountOptionMSNoDev:
			case MountOptionMSNoDirATime:
			case MountOptionMSNoATime:
			case MountOptionMSNoExec:
			case MountOptionMSNoSUID:
			case MountOptionMSRelatime:
			case MountOptionMSReadOnly:
			case MountOptionMSStrictATime:
			default:
				return fmt.Errorf("unsupported flag for remount operation: %s", opt)
			}
		}
		return nil
	}
	verifyBindMount := func(mountOpts []MountOption) error {
		bindMount := false
		for _, opt := range mountOpts {
			if opt == MountOptionMSBind {
				bindMount = true
			}
		}
		if !bindMount {
			return nil
		}
		for _, opt := range mountOpts {
			switch opt {
			case MountOptionMSRecursive:
			case MountOptionMSBind:
			default:
				return fmt.Errorf("unsupported flag for bind mount operation: %s", opt)
			}
		}
		return nil
	}
	verifyMountPropagation := func(mountOpts []MountOption) error {
		mountPropagation := false
		for _, opt := range mountOpts {
			switch opt {
			case MountOptionMSShared, MountOptionMSSlave, MountOptionMSPrivate, MountOptionMSUnBindable:
				mountPropagation = true
			}
		}
		if !mountPropagation {
			return nil
		}
		if len(mountOpts) > 2 {
			return fmt.Errorf("redundant/multiple mount propagation flags: %v", mountOpts)
		}
		for _, opt := range mountOpts {
			switch opt {
			case MountOptionMSRecursive:
			case MountOptionMSShared:
			case MountOptionMSSlave:
			case MountOptionMSPrivate:
			case MountOptionMSUnBindable:
			default:
				return fmt.Errorf("unsupported flag for bind mount operation: %s", opt)
			}
		}
		return nil
	}
	verifyFlags := func(mountOpts []MountOption) error {
		if err := verifyRemount(mountOpts); err != nil {
			return err
		}
		if err := verifyBindMount(mountOpts); err != nil {
			return err
		}
		if err := verifyMountPropagation(mountOpts); err != nil {
			return err
		}
		return nil
	}

	if err := verifyFlags(mountOpts); err != nil {
		return err
	}
	flags := uintptr(0)
	for _, opt := range mountOpts {
		switch opt {
		case MountOptionMSRemount:
			flags = flags | syscall.MS_REMOUNT
		case MountOptionMSBind:
			flags = flags | syscall.MS_BIND
		case MountOptionMSShared:
			flags = flags | syscall.MS_SHARED
		case MountOptionMSPrivate:
			flags = flags | syscall.MS_PRIVATE
		case MountOptionMSSlave:
			flags = flags | syscall.MS_SLAVE
		case MountOptionMSUnBindable:
			flags = flags | syscall.MS_UNBINDABLE
		case MountOptionMSMove:
			flags = flags | syscall.MS_MOVE
		case MountOptionMSDirSync:
			flags = flags | syscall.MS_DIRSYNC
		case MountOptionMSMandLock:
			flags = flags | syscall.MS_MANDLOCK
		case MountOptionMSNoATime:
			flags = flags | syscall.MS_NOATIME
		case MountOptionMSNoDev:
			flags = flags | syscall.MS_NODEV
		case MountOptionMSNoDirATime:
			flags = flags | syscall.MS_NODIRATIME
		case MountOptionMSNoExec:
			flags = flags | syscall.MS_NOEXEC
		case MountOptionMSNoSUID:
			flags = flags | syscall.MS_NOSUID
		case MountOptionMSReadOnly:
			flags = flags | syscall.MS_RDONLY
		case MountOptionMSRelatime:
			flags = flags | syscall.MS_RELATIME
		case MountOptionMSRecursive:
			flags = flags | syscall.MS_REC
		case MountOptionMSSilent:
			flags = flags | syscall.MS_SILENT
		case MountOptionMSStrictATime:
			flags = flags | syscall.MS_STRICTATIME
		case MountOptionMSSynchronous:
			flags = flags | syscall.MS_SYNCHRONOUS
		default:
			return fmt.Errorf("unsupported mount flag: %s", opt)
		}
	}

	klog.V(5).Infof("mounting %s at %s", source, target)
	return syscall.Mount(source, target, fsType, flags, strings.Join(superblockOpts, ","))
}

func SafeUnmount(target string, opts []UnmountOption) error {
	mountInfos, err := ProbeMounts()
	if err != nil {
		return err
	}

	mounted := false
	for _, mounts := range mountInfos {
		for _, mount := range mounts {
			if mount.MountPoint == target {
				mounted = true
				break
			}
		}
	}

	if !mounted {
		klog.V(3).Infof("drive already unmounted: %s", target)
		return nil
	}

	return Unmount(target, opts)
}

func SafeUnmountAll(path string, opts []UnmountOption) error {
	mountInfos, err := ProbeMounts()
	if err != nil {
		return err
	}

	major, minor, err := GetMajorMinor(path)
	if err != nil {
		return err
	}

	if mounts, found := mountInfos[fmt.Sprintf("%v:%v", major, minor)]; found {
		for _, mount := range mounts {
			if err := SafeUnmount(mount.MountPoint, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

func Unmount(target string, opts []UnmountOption) error {
	flags := 0
	for _, opt := range opts {
		switch opt {
		case UnmountOptionForce:
			flags = flags | syscall.MNT_FORCE
		case UnmountOptionDetach:
			flags = flags | syscall.MNT_DETACH
		case UnmountOptionExpire:
			flags = flags | syscall.MNT_EXPIRE
		default:
			return fmt.Errorf("Unsupport unmount flag: %s", opt)
		}
	}
	klog.V(5).Infof("unmounting %s", target)
	return syscall.Unmount(target, flags)
}

func ForceUnmount(target string) {
	err := syscall.Unmount(target, syscall.MNT_FORCE|syscall.MNT_DETACH)
	switch {
	case err == nil:
		klog.V(5).Infof("%v forcefully unmount successfully", target)
	case err != nil:
		klog.V(5).InfoS("unable to unmount", "err", err, "target", target)
	}
}
