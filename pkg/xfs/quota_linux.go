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

package xfs

import (
	"encoding/binary"
	"math"
	"os"
	"syscall"
	"unsafe"

	sha256 "github.com/minio/sha256-simd"
	"k8s.io/klog/v2"
)

const (
	// Refer below links for more information about these constants and their calculations.
	// - https://man7.org/linux/man-pages/man2/quotactl.2.html
	// - https://github.com/torvalds/linux/blob/master/include/uapi/linux/dqblk_xfs.h
	prjQuotaType = 2
	subCmdShift  = 8
	subCmdMask   = 0x00ff

	setQuotaCmd = 0x5804
	prjSetQuota = uintptr(setQuotaCmd<<subCmdShift | prjQuotaType&subCmdMask)

	getQuotaCmd = 0x5803
	prjGetQuota = uintptr(getQuotaCmd<<subCmdShift | prjQuotaType&subCmdMask)

	fsDiskQuotaVersion  = 1
	xfsProjectQuotaFlag = 2
	fieldMaskBHard      = 8
	fieldMaskBSoft      = 4
	blockSize           = 512

	fsGetAttr          = 0x801c581f // FS_IOC_FSGETXATTR
	fsSetAttr          = 0x401c5820 // FS_IOC_FSSETXATTR
	flagProjectInherit = 0x00000200
)

// Refer below link for more information about this structure.
// - https://man7.org/linux/man-pages/man2/quotactl.2.html
type fsDiskQuota struct {
	version         int8    // Version of this structure
	flags           int8    // XFS_{USER,PROJ,GROUP}_QUOTA
	fieldmask       uint16  // Field specifier
	id              uint32  // User, project, or group ID
	hardLimitBlocks uint64  // Absolute limit on disk blocks
	softLimitBlocks uint64  // Preferred limit on disk blocks
	_               uint64  // hardLimitInodes: Maximum allocated inodes
	_               uint64  // softLimitInodes: Preferred inode limit
	blocksCount     uint64  // disk blocks owned by the project/user/group
	_               uint64  // inodesCount: inodes owned by the project/user/group
	_               int32   // inodeTimer: Zero if within inode limits, If not, we refuse service
	_               int32   // blocksTimer: Similar to above; for disk blocks
	_               uint16  // inodeWarnings: warnings issued with respect to number of inodes
	_               uint16  // blockWarnings: warnings issued with respect to disk blocks
	_               int32   // padding2: Padding - for future use
	_               uint64  // rtbHardLimit: Absolute limit on realtime (RT) disk blocks
	_               uint64  // rtbSoftLimit: Preferred limit on RT disk blocks
	_               uint64  // rtbCount: realtime blocks owned
	_               int32   // rtbTimer: Similar to above; for RT disk blocks
	_               uint16  // rtbWarnings: warnings issued with respect to RT disk blocks
	_               int16   // padding3: Padding - for future use
	_               [8]byte // padding4: Yet more padding
}

type fsXAttr struct {
	fsXXFlags uint32
	_         uint32 // fsXExtSize
	_         uint32 // fsXNextents
	fsXProjID uint32
	_         uint32  // fsXCowextSize
	_         [8]byte // fsXPad
}

func getProjectIDHash(id string) uint32 {
	hash := sha256.Sum256([]byte(id))
	return binary.LittleEndian.Uint32(hash[:8])
}

func getQuota(device, volumeID string) (*Quota, error) {
	deviceNamePtr, err := syscall.BytePtrFromString(device)
	if err != nil {
		return nil, err
	}
	projectID := int(getProjectIDHash(volumeID))

	result := &fsDiskQuota{}
	_, _, errno := syscall.RawSyscall6(
		syscall.SYS_QUOTACTL,
		prjGetQuota,
		uintptr(unsafe.Pointer(deviceNamePtr)),
		uintptr(projectID),
		uintptr(unsafe.Pointer(result)),
		0,
		0,
	)
	if errno != 0 {
		return nil, os.NewSyscallError("quotactl", errno)
	}

	return &Quota{
		HardLimit:    result.hardLimitBlocks * blockSize,
		SoftLimit:    result.softLimitBlocks * blockSize,
		CurrentSpace: result.blocksCount * blockSize,
	}, nil
}

func setProjectID(path string, projectID uint32) error {
	targetDir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer targetDir.Close()

	var fsx fsXAttr
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		targetDir.Fd(),
		fsGetAttr,
		uintptr(unsafe.Pointer(&fsx)),
	)
	if errno != 0 {
		return os.NewSyscallError("FS_IOC_FSGETXATTR", errno)
	}

	fsx.fsXProjID = projectID
	fsx.fsXXFlags |= uint32(flagProjectInherit)
	_, _, errno = syscall.Syscall(
		syscall.SYS_IOCTL,
		targetDir.Fd(),
		fsSetAttr,
		uintptr(unsafe.Pointer(&fsx)),
	)
	if errno != 0 {
		return os.NewSyscallError("FS_IOC_FSSETXATTR", errno)
	}

	return nil
}

func setProjectQuota(device string, projectID uint32, quota Quota) error {
	hardLimitBlocks := uint64(math.Ceil(float64(quota.HardLimit) / blockSize))
	softLimitBlocks := uint64(math.Ceil(float64(quota.SoftLimit) / blockSize))

	fsQuota := &fsDiskQuota{
		version:         int8(fsDiskQuotaVersion),
		flags:           int8(xfsProjectQuotaFlag),
		fieldmask:       uint16(fieldMaskBHard | fieldMaskBSoft),
		id:              projectID,
		hardLimitBlocks: hardLimitBlocks,
		softLimitBlocks: softLimitBlocks,
	}

	deviceNamePtr, err := syscall.BytePtrFromString(device)
	if err != nil {
		return err
	}

	_, _, errno := syscall.Syscall6(
		syscall.SYS_QUOTACTL,
		prjSetQuota,
		uintptr(unsafe.Pointer(deviceNamePtr)),
		uintptr(projectID),
		uintptr(unsafe.Pointer(fsQuota)),
		0,
		0,
	)
	if errno != 0 {
		return os.NewSyscallError("quotactl", errno)
	}

	return nil
}

func setQuota(device, path, volumeID string, quota Quota) error {
	if info, err := getQuota(device, volumeID); err == nil {
		klog.V(3).InfoS("Quota is already set", "Device", device, "Path", path, "VolumeID", volumeID, "ProjectID", "HardLimitSet", info.HardLimit, "HardLimit", info.HardLimit)
		return nil
	}

	projectID := getProjectIDHash(volumeID)
	if err := setProjectID(path, projectID); err != nil {
		klog.ErrorS(err, "unable to set project ID", "Device", device, "Path", path)
		return err
	}

	if err := setProjectQuota(device, projectID, quota); err != nil {
		klog.ErrorS(err, "unable to set quota", "Device", device, "Path", path, "Limit", quota.HardLimit)
		return err
	}

	klog.V(3).InfoS("SetQuota succeeded", "Device", device, "Path", path, "VolumeID", volumeID, "ProjectID", projectID, "HardLimit", quota.HardLimit)
	return nil
}
