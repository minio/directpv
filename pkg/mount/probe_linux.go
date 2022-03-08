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

package mount

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func probe(filename string) (map[string][]MountInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	mountPointsMap := map[string][]MountInfo{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		// Refer /proc/[pid]/mountinfo section in https://man7.org/linux/man-pages/man5/proc.5.html
		// to know about this logic.
		tokens := strings.Fields(strings.TrimSpace(s))
		if len(tokens) < 8 {
			return nil, fmt.Errorf("unknown format %v", strings.TrimSpace(s))
		}

		majorMinor := tokens[2]
		mountPoint := tokens[4]
		mountOptions := strings.Split(tokens[5], ",")
		sort.Strings(mountOptions)

		// Skip mount tags.
		var i int
		for i = 6; i < len(tokens); i++ {
			if tokens[i] == "-" {
				i++
				break
			}
		}

		fsType := tokens[i]
		fsSubType := ""
		if fsTokens := strings.SplitN(tokens[i], ".", 2); len(fsTokens) == 2 {
			fsType = fsTokens[0]
			fsSubType = fsTokens[1]
		}

		mountPointsMap[majorMinor] = append(
			mountPointsMap[majorMinor],
			MountInfo{
				MajorMinor:   majorMinor,
				MountPoint:   mountPoint,
				MountOptions: mountOptions,
				fsType:       fsType,
				fsSubType:    fsSubType,
			},
		)
	}

	return mountPointsMap, nil
}
