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

package node

import (
	"fmt"
	"path/filepath"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/matcher"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
)

func checkDrive(drive *directcsi.DirectCSIDrive, volumeID string, probeMounts func() (map[string][]mount.Info, error)) error {
	if drive.Status.DriveStatus != directcsi.DriveStatusInUse {
		return fmt.Errorf("drive %v is not in InUse state", drive.Name)
	}

	finalizer := directcsi.DirectCSIDriveFinalizerPrefix + volumeID
	if !matcher.StringIn(drive.Finalizers, finalizer) {
		return fmt.Errorf("drive %v does not have volume finalizer %v", drive.Name, finalizer)
	}

	mounts, err := probeMounts()
	if err != nil {
		return err
	}

	majorMinor := fmt.Sprintf("%v:%v", drive.Status.MajorNumber, drive.Status.MinorNumber)
	mountInfos, found := mounts[majorMinor]
	if !found {
		return fmt.Errorf("mount information not found for major/minor %v of drive %v", majorMinor, drive.Name)
	}

	mountPoint := filepath.Join(sys.MountRoot, drive.Status.FilesystemUUID)
	for _, mountInfo := range mountInfos {
		if mountInfo.MountPoint == mountPoint {
			return nil
		}
	}

	return fmt.Errorf("drive %v is not mounted at mount point %v", drive.Name, mountPoint)
}

func checkStagingTargetPath(stagingPath string, probeMounts func() (map[string][]mount.Info, error)) error {
	mounts, err := probeMounts()
	if err != nil {
		return err
	}

	for _, mountInfos := range mounts {
		for _, mountInfo := range mountInfos {
			if mountInfo.MountPoint == stagingPath {
				return nil
			}
		}
	}

	return fmt.Errorf("stagingPath %v is not mounted", stagingPath)
}
