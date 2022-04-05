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
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
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

func getDriveUUID(nodeID string, device *sys.Device) string {
	data := []byte(
		strings.Join(
			[]string{
				nodeID,
				device.WWID,
				device.UeventSerial,
				device.DMUUID,
				device.SerialLong,
				device.PCIPath,
				strconv.Itoa(device.Partition),
			},
			"",
		),
	)
	h := sha256.Sum256(data)
	return uuid.UUID{
		h[0],
		h[1],
		h[2],
		h[3],
		h[4],
		h[5],
		h[6],
		h[7],
		h[8],
		h[9],
		h[10],
		h[11],
		h[12],
		h[13],
		h[14],
		h[15],
	}.String()
}
