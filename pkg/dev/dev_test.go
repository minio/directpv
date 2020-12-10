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
	"testing"
)

func TestGetPartitionPath(t1 *testing.T) {
	testCases := []struct {
		name       string
		partPaths  []string
		partNum    int
		resultPath string
	}{
		{
			name:       "test1",
			partPaths:  []string{"/dev/nvme0n1p2", "/dev/nvme0n1p1"},
			partNum:    1,
			resultPath: "/dev/nvme0n1p1",
		},
		{
			name:       "test2",
			partPaths:  []string{"/dev/nvme0n1p2", "/dev/nvme0n1p1"},
			partNum:    2,
			resultPath: "/dev/nvme0n1p2",
		},
		{
			name:       "test3",
			partPaths:  []string{"/dev/nvme0n1p2", "/dev/nvme0n1p1"},
			partNum:    3,
			resultPath: "",
		},
		{
			name:       "test4",
			partPaths:  []string{"/dev/nvme0n1p2", "/dev/nvme0n1p33"},
			partNum:    3,
			resultPath: "",
		},
		{
			name:       "test5",
			partPaths:  []string{"/dev/nvme0n1p2", "/dev/nvme0n1p22"},
			partNum:    22,
			resultPath: "/dev/nvme0n1p22",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			resPath := GetPartitionPath(tt.partPaths, tt.partNum)
			if resPath != tt.resultPath {
				t1.Errorf("Test case name %s: Expected path = %s, got %v", tt.name, tt.resultPath, resPath)
			}
		})
	}

}
