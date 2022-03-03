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

package sys

import (
	"testing"
)

func TestGetBlockFile(t1 *testing.T) {

	testCases := []struct {
		name      string
		devName   string
		blockFile string
	}{
		{
			name:      "test1",
			devName:   "/var/lib/direct-csi/devices/xvdb",
			blockFile: "/var/lib/direct-csi/devices/xvdb",
		},
		{
			name:      "test2",
			devName:   "/var/lib/direct-csi/devices/xvdb-part-1",
			blockFile: "/var/lib/direct-csi/devices/xvdb-part-1",
		},
		{
			name:      "test3",
			devName:   "/dev/xvdb",
			blockFile: "/var/lib/direct-csi/devices/xvdb",
		},
		{
			name:      "test4",
			devName:   "/dev/xvdb3",
			blockFile: "/var/lib/direct-csi/devices/xvdb-part-3",
		},
		{
			name:      "test5",
			devName:   "/dev/xvdb15",
			blockFile: "/var/lib/direct-csi/devices/xvdb-part-15",
		},
		{
			name:      "test6",
			devName:   "/dev/nvmen1p4",
			blockFile: "/var/lib/direct-csi/devices/nvmen1p-part-4",
		},
		{
			name:      "test7",
			devName:   "/dev/nvmen12p11",
			blockFile: "/var/lib/direct-csi/devices/nvmen12p-part-11",
		},
		{
			name:      "test8",
			devName:   "/var/lib/direct-csi/devices/loop0",
			blockFile: "/var/lib/direct-csi/devices/loop0",
		},
		{
			name:      "test9",
			devName:   "/var/lib/direct-csi/devices/loop-part-12",
			blockFile: "/var/lib/direct-csi/devices/loop-part-12",
		},
		{
			name:      "test10",
			devName:   "loop0",
			blockFile: "/var/lib/direct-csi/devices/loop0",
		},
		{
			name:      "test11",
			devName:   "loop12",
			blockFile: "/var/lib/direct-csi/devices/loop-part-12",
		},
		{
			name:      "test12",
			devName:   "/dev/loop3",
			blockFile: "/var/lib/direct-csi/devices/loop-part-3",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			blockFile := getBlockFile(tt.devName)
			if blockFile != tt.blockFile {
				t1.Errorf("Test case name %s: Expected block file = (%s) got: %s", tt.name, tt.blockFile, blockFile)
			}
		})
	}

}
