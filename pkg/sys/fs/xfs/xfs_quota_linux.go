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

package xfs

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"syscall"
	"unsafe"

	simd "github.com/minio/sha256-simd"
	"k8s.io/klog/v2"
)

type XFSQuota struct {
	BlockFile string
	Path      string
	VolumeID  string
}

type XFSVolumeStats struct {
	AvailableBytes int64
	TotalBytes     int64
	UsedBytes      int64
}

type Dqblk struct {
	// Definition since Linux 2.4.22
	DqbBHardlimit uint64 // Absolute limit on disk quota blocks alloc
	DqbBSoftlimit uint64 // Preferred limit on disk quota blocks
	DqbCurSpace   uint64 // Current occupied space (in bytes)
	DqbIHardlimit uint64 // Maximum number of allocated inodes
	DqbISoftlimit uint64 // Preferred inode limit
	DqbCurInodes  uint64 // Current number of allocated inodes
	DqbBTime      uint64 // Time limit for excessive disk use
	DqbITime      uint64 // Time limit for excessive files
	DqbValid      uint32 // Bit mask of QIF_* constantss
}

type FSXAttr struct {
	FSXXFlags     uint32
	FSXExtSize    uint32
	FSXNextents   uint32
	FSXProjID     uint32
	FSXCowextSize uint32
	FSXPad        [8]byte
}

func getProjectIDHash(id string) uint32 {
	h := simd.Sum256([]byte(id))
	return binary.LittleEndian.Uint32(h[:8])
}

func (xfsq *XFSQuota) setProjectID(projectID uint32) error {

	targetDir, err := os.Open(xfsq.Path)
	if err != nil {
		return fmt.Errorf("could not open %v: %v", xfsq.Path, err)
	}
	defer targetDir.Close()

	var fsx FSXAttr
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		targetDir.Fd(),
		FS_IOC_FSGETXATTR,
		uintptr(unsafe.Pointer(&fsx))); errno != 0 {
		return fmt.Errorf("failed to execute GETFSXAttrs. path: %v volume: %v error: %v", xfsq.Path, xfsq.VolumeID, errno)
	}

	fsx.FSXProjID = uint32(projectID)
	fsx.FSXXFlags |= uint32(FlagProjectInherit)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		targetDir.Fd(),
		FS_IOC_FSSETXATTR,
		uintptr(unsafe.Pointer(&fsx))); errno != 0 {
		return fmt.Errorf("failed to execute SETFSXAttrs. path: %v volume: %v projectID: %v error: %v", xfsq.Path, xfsq.VolumeID, fsx.FSXProjID, errno)
	}

	return nil
}

func (xfsq *XFSQuota) setQuota(maxBytes uint64, projID uint32) error {

	bytesLimitBlocks := uint64(math.Ceil(float64(maxBytes) / float64(BlockSize)))
	quota := &Dqblk{
		DqbBHardlimit: bytesLimitBlocks,
		DqbBSoftlimit: bytesLimitBlocks,
		DqbValid:      FlagBLimitsValid,
	}

	deviceNamePtr, err := syscall.BytePtrFromString(xfsq.BlockFile)
	if err != nil {
		return err
	}
	if _, _, errno := syscall.Syscall6(syscall.SYS_QUOTACTL,
		uintptr(setPrjQuotaSubCmd),
		uintptr(unsafe.Pointer(deviceNamePtr)),
		uintptr(projID),
		uintptr(unsafe.Pointer(quota)),
		0,
		0); errno != syscall.Errno(0) {
		return fmt.Errorf("failed to set quota: %w", err)
	}

	return nil
}

// SetQuota creates a projectID and sets the hardlimit for the path
func (xfsq *XFSQuota) SetQuota(ctx context.Context, limit int64) error {
	_, err := xfsq.GetQuota()
	// this means quota has already been set
	if err == nil {
		return nil
	}

	projectID := getProjectIDHash(xfsq.VolumeID)
	if err := xfsq.setProjectID(projectID); err != nil {
		klog.Errorf("could not set projectID err=%v", err)
		return err
	}

	klog.V(3).InfoS("Setting projectquota",
		"VolumeID", xfsq.VolumeID,
		"ProjectID", projectID,
		"Path", xfsq.Path,
		"limit", limit)
	if err := xfsq.setQuota(uint64(limit), projectID); err != nil {
		klog.Errorf("could not setquota err=%v", err)
		return err
	}
	klog.V(3).InfoS("Successfully set projectquota",
		"VolumeID", xfsq.VolumeID,
		"ProjectID", projectID)
	return nil
}

func (xfsq *XFSQuota) GetQuota() (result *Dqblk, err error) {
	result = &Dqblk{}
	var deviceNamePtr *byte
	if deviceNamePtr, err = syscall.BytePtrFromString(xfsq.BlockFile); err != nil {
		return
	}
	projectID := int(getProjectIDHash(xfsq.VolumeID))

	if _, _, errno := syscall.RawSyscall6(syscall.SYS_QUOTACTL,
		uintptr(getPrjQuotaSubCmd),
		uintptr(unsafe.Pointer(deviceNamePtr)),
		uintptr(projectID),
		uintptr(unsafe.Pointer(result)),
		0,
		0); errno != 0 {
		err = os.NewSyscallError("quotactl", errno)
	}

	return
}
