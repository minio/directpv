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

package dev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Mount struct {
	MountPoint        string   `json:"mountPoint,omitempty"`
	MountFlags        []string `json:"mountFlags,omitempty"`
	MountRoot         string   `json:"mountRoot,omitempty"`
	MountID           uint32   `json:"mountID,omitempty"`
	ParentID          uint32   `json:"parentID,omitempty"`
	MountSource       string   `json:"mountSource,omitempty"`
	SuperblockOptions []string `json:"superblockOptions,omitempty"`
	FSType            FSType   `json:"fsType,omitempty"`
	OptionalFields    []string `json:"optionalFields,omitempty"`
}

func ProbeMounts(procfs string, devName string, partitionNum uint) ([]Mount, error) {
	mountinfoFile := filepath.Join(procfs, "1", "mountinfo")
	f, err := os.Open(mountinfoFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mounts := []Mount{}
	fbuf := bufio.NewReader(f)

	for {
		line, err := fbuf.ReadString(byte('\n'))
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		devName := filepath.Join(DevRoot, devName)
		// usual naming scheme
		if strings.Contains(line, devName) {
			parts := strings.SplitN(line, "-", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid format of %s 1", mountinfoFile)
			}
			firstParts := strings.Fields(strings.TrimSpace(parts[0]))
			if len(firstParts) < 7 {
				return nil, fmt.Errorf("invalid format of %s 2", mountinfoFile)
			}
			mID, err := strconv.ParseUint(firstParts[0], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid format of %s 3", mountinfoFile)
			}
			mountID := uint32(mID)

			pID, err := strconv.ParseUint(firstParts[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid format of %s 4", mountinfoFile)
			}
			parentID := uint32(pID)

			mountRoot := firstParts[3]
			mountPoint := firstParts[4]
			mountOptions := firstParts[5]
			optionalFields := firstParts[6:]
			mountFlags := strings.Split(mountOptions, ",")

			secondParts := strings.Fields(strings.TrimSpace(parts[1]))
			if len(secondParts) < 3 {
				return nil, fmt.Errorf("invalid format of %s 5", mountinfoFile)
			}

			fsType := FSType(secondParts[0])
			mountSource := secondParts[1]
			if !strings.HasSuffix(mountSource, fmt.Sprintf("%d", partitionNum)) {
				continue
			}
			superblockOptions := strings.Split(secondParts[2], ",")

			mounts = append(mounts, Mount{
				MountPoint:        mountPoint,
				MountFlags:        mountFlags,
				MountRoot:         mountRoot,
				MountID:           mountID,
				ParentID:          parentID,
				MountSource:       mountSource,
				SuperblockOptions: superblockOptions,
				FSType:            fsType,
				OptionalFields:    optionalFields,
			})
		}
	}
	return mounts, nil
}
