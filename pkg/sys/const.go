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

const (
	// HostDevRoot is "/dev" directory.
	HostDevRoot = "/dev"

	// MountRoot is "/var/lib/direct-csi/mnt" directory.
	MountRoot = "/var/lib/direct-csi/mnt"

	// DirectCSIDevRoot is "/var/lib/direct-csi/devices" directory.
	DirectCSIDevRoot = "/var/lib/direct-csi/devices"

	// DirectCSIPartitionInfix is partition infix value.
	DirectCSIPartitionInfix = "-part-"

	// HostPartitionInfix is host infix value.
	HostPartitionInfix = "p"
)

// FSType is filesystem type.
type FSType string

const (
	// FSTypeXFS is XFS filesystem type.
	FSTypeXFS FSType = "xfs"
)

// MountOption denotes device mount options.
type MountOption string

// Mount options.
const (
	MountOptionMSRemount     MountOption = "remount"
	MountOptionMSBind        MountOption = "bind"
	MountOptionMSShared      MountOption = "shared"
	MountOptionMSPrivate     MountOption = "private"
	MountOptionMSSlave       MountOption = "slave"
	MountOptionMSUnBindable  MountOption = "unbindable"
	MountOptionMSMove        MountOption = "move"
	MountOptionMSDirSync     MountOption = "dirsync"
	MountOptionMSMandLock    MountOption = "mand"
	MountOptionMSNoATime     MountOption = "noatime"
	MountOptionMSNoDev       MountOption = "nodev"
	MountOptionMSNoDirATime  MountOption = "nodiratime"
	MountOptionMSNoExec      MountOption = "noexec"
	MountOptionMSNoSUID      MountOption = "nosuid"
	MountOptionMSReadOnly    MountOption = "ro"
	MountOptionMSRelatime    MountOption = "relatime"
	MountOptionMSRecursive   MountOption = "recursive"
	MountOptionMSSilent      MountOption = "silent"
	MountOptionMSStrictATime MountOption = "strictatime"
	MountOptionMSSynchronous MountOption = "sync"
)

// UnmountOption denotes device unmount options.
type UnmountOption string

// Unmount options.
const (
	UnmountOptionForce  UnmountOption = "force"
	UnmountOptionDetach UnmountOption = "detach"
	UnmountOptionExpire UnmountOption = "expire"
)
