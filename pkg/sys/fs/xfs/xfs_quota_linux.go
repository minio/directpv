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
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"

	simd "github.com/minio/sha256-simd"
	"k8s.io/klog"
)

type XFSQuota struct {
	BlockFile string
	Path      string
	ProjectID string
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

func getProjectIDHash(id string) uint32 {
	h := simd.Sum256([]byte(id))
	return binary.LittleEndian.Uint32(h[:8])
}

// SetQuota creates a projectID and sets the hardlimit for the path
func (xfsq *XFSQuota) SetQuota(ctx context.Context, limit int64) error {
	_, err := xfsq.GetQuota()
	// this means quota has already been set
	if err == nil {
		return nil
	}

	limitInStr := strconv.FormatInt(limit, 10)
	projID := getProjectIDHash(xfsq.ProjectID)
	pid := strconv.FormatUint(uint64(projID), 10)

	klog.V(3).Infof("setting prjquota proj_id=%s path=%s", pid, xfsq.Path)

	cmd := exec.CommandContext(ctx, "xfs_quota", "-x", "-c", fmt.Sprintf("project -d 0 -s -p %s %s", xfsq.Path, pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("could not set prjquota proj_id=%s path=%s err=%v", pid, xfsq.Path, err)
		return fmt.Errorf("SetQuota failed for %s with error: (%v), output: (%s)", xfsq.ProjectID, err, out)
	}

	cmd = exec.CommandContext(ctx, "xfs_quota", "-x", "-c", fmt.Sprintf("limit -p bhard=%s %s", limitInStr, pid), xfsq.Path)
	out, err = cmd.CombinedOutput()
	if err != nil {
		klog.Errorf("could not set prjquota proj_id=%s path=%s err=%v", pid, xfsq.Path, err)
		return fmt.Errorf("xfs_quota failed with error: %v, output: %s", err, out)
	}
	klog.V(3).Infof("prjquota set successfully proj_id=%s path=%s", pid, xfsq.Path)

	return nil
}

func (xfsq *XFSQuota) GetQuota() (result *Dqblk, err error) {
	result = &Dqblk{}
	var deviceNamePtr *byte
	if deviceNamePtr, err = syscall.BytePtrFromString(xfsq.BlockFile); err != nil {
		return
	}
	projectID := int(getProjectIDHash(xfsq.ProjectID))

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
