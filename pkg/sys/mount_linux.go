/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package sys

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/golang/glog"
)

func SafeMount(source, target, fsType string, mountOpts []MountOption, superblockOpts []string) error {
	mounts, err := ProbeMountInfo()
	if err != nil {
		return err
	}

	for _, m := range mounts {
		// idempotency check
		if m.Mountpoint == target {
			if len(m.MountFlags) != len(mountOpts) {
				break
			}

			allMFlagsFound := true
			for _, opt := range mountOpts {
				optFound := false
				for _, mflag := range m.MountFlags {
					if mflag == string(opt) {
						optFound = true
						break
					}
				}
				if !optFound {
					allMFlagsFound = false
					break
				}
			}
			if allMFlagsFound {
				glog.V(3).Infof("drive already mounted: %s", target)
				// if already mounted at the same position with same flags
				return nil
			}
			break
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
			case MountOptionMSShared:
				fallthrough
			case MountOptionMSSlave:
				fallthrough
			case MountOptionMSPrivate:
				fallthrough
			case MountOptionMSUnBindable:
				mountPropagation = true
				break
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
			return fmt.Errorf("Unsupported mount flag: %s", opt)
		}
	}

	glog.V(5).Infof("mounting %s at %s", source, target)
	return syscall.Mount(source, target, fsType, flags, strings.Join(superblockOpts, ","))
}

func SafeUnmount(target string, opts []UnmountOption) error {
	mounts, err := ProbeMountInfo()
	if err != nil {
		return err
	}

	targetMountFound := false
	for _, m := range mounts {
		// idempotency check
		if m.Mountpoint == target {
			targetMountFound = true
			break
		}
	}

	// if no mounts were found at the given path
	if !targetMountFound {
		glog.V(3).Infof("drive already unmounted: %s", target)
		return nil
	}
	return Unmount(target, opts)

}

func SafeUnmountAll(drivePath string, opts []UnmountOption) error {
	mounts, err := ProbeMountInfo()
	if err != nil {
		return err
	}

	for _, m := range mounts {
		if getBlockFile(m.MountSource) == getBlockFile(drivePath) {
			if err := SafeUnmount(m.Mountpoint, opts); err != nil {
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
	glog.V(5).Infof("unmounting %s", target)
	return syscall.Unmount(target, flags)
}
