/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
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

package utils

type FSType string

const (
	FSTypeXFS  FSType = "xfs"
	FSTypeEXT4        = "ext4"
)

// Path constants
const (
	DirectCSIRoot = "/var/lib/direct-csi"
	MountRoot     = "/var/lib/direct-csi/mnt"
	DevRoot       = "/var/lib/direct-csi/devices"
)

// Mount options
type MountOption string

const (
	MountOptionMSRemount     MountOption = "remount"
	MountOptionMSBind                    = "bind"
	MountOptionMSShared                  = "shared"
	MountOptionMSPrivate                 = "private"
	MountOptionMSSlave                   = "slave"
	MountOptionMSUnBindable              = "unbindable"
	MountOptionMSMove                    = "move"
	MountOptionMSDirSync                 = "dirsync"
	MountOptionMSMandLock                = "mand"
	MountOptionMSNoATime                 = "noatime"
	MountOptionMSNoDev                   = "nodev"
	MountOptionMSNoDirATime              = "nodiratime"
	MountOptionMSNoExec                  = "noexec"
	MountOptionMSNoSUID                  = "nosuid"
	MountOptionMSReadOnly                = "ro"
	MountOptionMSRelatime                = "relatime"
	MountOptionMSRecursive               = "recursive"
	MountOptionMSSilent                  = "silent"
	MountOptionMSStrictATime             = "strictatime"
	MountOptionMSSynchronous             = "sync"
)

// Unmount options
type UnmountOption string

const (
	UnmountOptionForce    UnmountOption = "force"
	UnmountOptionDetach                 = "detach"
	UnmountOptionExpire                 = "expire"
)
