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

package fs

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

func NewQuotaer(targetPath, vID, blockFile string) Quotaer {
	return &FSQuota{
		Path:      targetPath,
		VolumeID:  vID,
		BlockFile: blockFile,
	}
}

func getProjectIDHash(id string) uint32 {
	h := simd.Sum256([]byte(id))
	return binary.LittleEndian.Uint32(h[:8])
}

func (fsq FSQuota) GetBlockFile() string {
	return fsq.BlockFile
}

func (fsq FSQuota) GetPath() string {
	return fsq.Path
}

func (fsq FSQuota) GetVolumeID() string {
	return fsq.VolumeID
}

func (q *FSQuota) SetProjectID(projectID uint32) error {

	targetDir, err := os.Open(q.GetPath())
	if err != nil {
		return fmt.Errorf("could not open %v: %v", q.GetPath(), err)
	}
	defer targetDir.Close()

	var fsx FSXAttr
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		targetDir.Fd(),
		FS_IOC_FSGETXATTR,
		uintptr(unsafe.Pointer(&fsx))); errno != 0 {
		return fmt.Errorf("failed to execute GETFSXAttrs. path: %v volume: %v error: %v", q.GetPath(), q.GetVolumeID(), errno)
	}

	fsx.FSXProjID = uint32(projectID)
	fsx.FSXXFlags |= uint32(FlagProjectInherit)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		targetDir.Fd(),
		FS_IOC_FSSETXATTR,
		uintptr(unsafe.Pointer(&fsx))); errno != 0 {
		return fmt.Errorf("failed to execute SETFSXAttrs. path: %v volume: %v projectID: %v error: %v", q.GetPath(), q.GetVolumeID(), fsx.FSXProjID, errno)
	}

	return nil
}

func (fsq *FSQuota) SetProjectQuota(maxBytes uint64, projID uint32) error {

	bytesLimitBlocks := uint64(math.Ceil(float64(maxBytes) / float64(BlockSize)))
	quota := &Dqblk{
		DqbBHardlimit: bytesLimitBlocks,
		DqbBSoftlimit: bytesLimitBlocks,
		DqbValid:      FlagBLimitsValid,
	}

	deviceNamePtr, err := syscall.BytePtrFromString(fsq.GetBlockFile())
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

func SetFSQuota(ctx context.Context, fsq Quotaer, limit int64) error {
	_, err := fsq.GetQuota()
	// this means quota has already been set
	if err == nil {
		return nil
	}

	projectID := getProjectIDHash(fsq.GetVolumeID())
	if err := fsq.SetProjectID(projectID); err != nil {
		klog.Errorf("could not set projectID err=%v", err)
		return err
	}

	klog.V(3).InfoS("Setting projectquota",
		"VolumeID", fsq.GetVolumeID(),
		"ProjectID", projectID,
		"Path", fsq.GetPath(),
		"limit", limit)
	if err := fsq.SetProjectQuota(uint64(limit), projectID); err != nil {
		klog.Errorf("could not setquota err=%v", err)
		return err
	}
	klog.V(3).InfoS("Successfully set projectquota",
		"VolumeID", fsq.GetVolumeID(),
		"ProjectID", projectID)
	return nil
}

// SetQuota creates a projectID and sets the hardlimit for the path
func (fsq *FSQuota) SetQuota(ctx context.Context, limit int64) error {
	return SetFSQuota(ctx, fsq, limit)
}

func (fsq *FSQuota) GetQuota() (result *Dqblk, err error) {
	result = &Dqblk{}
	var deviceNamePtr *byte
	if deviceNamePtr, err = syscall.BytePtrFromString(fsq.GetBlockFile()); err != nil {
		return
	}
	projectID := int(getProjectIDHash(fsq.GetVolumeID()))

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
