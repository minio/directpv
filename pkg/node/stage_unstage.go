// This file is part of MinIO Direct CSI
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

package node

import (
	"context"
	"k8s.io/utils/mount"
	"os"
	"path/filepath"

	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
)

func StageVolume(ctx context.Context, directCSIDrive *direct_csi.DirectCSIDrive, stagingPath string, volumeID string) (string, error) {
	hostPath := filepath.Join(directCSIDrive.Mountpoint, volumeID)
	if err := os.MkdirAll(hostPath, 0755); err != nil {
		return "", err
	}

	if err := os.MkdirAll(stagingPath, 0755); err != nil {
		return "", err
	}

	if _, err := os.Lstat(stagingPath); err != nil {
		return "", err
	}

	mounter := mount.New("")

	shouldBindMount := true
	mountPoints, mntErr := mounter.List()
	if mntErr != nil {
		return "", mntErr
	}
	for _, mp := range mountPoints {
		abPath, _ := filepath.Abs(mp.Path)
		if stagingPath == abPath {
			shouldBindMount = false
			break
		}
	}

	if shouldBindMount {
		if err := mounter.Mount(hostPath, stagingPath, "", []string{"bind"}); err != nil {
			return "", err
		}
	}

	return hostPath, nil
}

func UnstageVolume(ctx context.Context, stagingPath string) error {
	if _, err := os.Lstat(stagingPath); err != nil {
		return err
	}

	mounter := mount.New("")
	return mounter.Unmount(stagingPath)
}
