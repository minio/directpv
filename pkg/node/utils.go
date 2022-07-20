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

	"github.com/minio/directpv/pkg/mount"
)

func checkStagingTargetPath(stagingPath string, probeMounts func() (map[string][]mount.MountInfo, error)) error {
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
