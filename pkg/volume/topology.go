// This file is part of MinIO Kubernetes Cloud
// Copyright (c) 2020 MinIO, Inc.
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

package volume

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/minio/minio/pkg/disk"
)

var (
	dt = &driveTopology{}
)

// DriveInfo captures information about each drive
type DriveInfo struct {
	MountPath string
	DrivePath string
	SysInfo   disk.Info
}

// driveTopology captures the information regarding
// the storage topology.
type driveTopology struct {
	LastAssigned int64
	Drives       []DriveInfo
}

// InitializeDrives initialize all the drives presented in storage topology
func InitializeDrives(drives []DriveInfo) {
	dt.Drives = drives
	dt.LastAssigned = -1
}

// Provision provisions a new volume by picking a new drive in round robin fashion
func Provision(volumeID string) (string, error) {
	if len(dt.Drives) == 0 {
		return "", fmt.Errorf("no drives present in storage topology for direct CSI")
	}

	next := int(atomic.LoadInt64(&dt.LastAssigned)+1) % len(dt.Drives)
	nextDrive := dt.Drives[next]

	glog.V(15).Infof("[%s] using direct storage: Drives[%d] = %v", volumeID, next, nextDrive)

	if err := os.MkdirAll(filepath.Join(nextDrive.MountPath, volumeID), 0755); err != nil {
		return "", err
	}

	atomic.StoreInt64(&dt.LastAssigned, int64(next))
	return filepath.Join(nextDrive.MountPath, volumeID), nil
}

// Unprovision - not used, perhaps in future used for decomission.
func Unprovision(path string) error {
	return os.RemoveAll(path)
}
