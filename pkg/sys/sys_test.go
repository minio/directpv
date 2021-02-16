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

func TestGetRootBlockFile(t1 *testing.T) {

	testCases := []struct {
		name     string
		devName  string
		rootFile string
	}{
		{
			name:     "test1",
			devName:  "/dev/xvdb",
			rootFile: "/dev/xvdb",
		},
		{
			name:     "test2",
			devName:  "/dev/xvdb1",
			rootFile: "/dev/xvdb1",
		},
		{
			name:     "test3",
			devName:  "/var/lib/direct-csi/devices/xvdb",
			rootFile: "/dev/xvdb",
		},
		{
			name:     "test4",
			devName:  "/var/lib/direct-csi/devices/xvdb-part-3",
			rootFile: "/dev/xvdb3",
		},
		{
			name:     "test5",
			devName:  "/var/lib/direct-csi/devices/xvdb-part-15",
			rootFile: "/dev/xvdb15",
		},
		{
			name:     "test6",
			devName:  "/var/lib/direct-csi/devices/nvmen1p-part-4",
			rootFile: "/dev/nvmen1p4",
		},
		{
			name:     "test7",
			devName:  "/var/lib/direct-csi/devices/nvmen12p-part-11",
			rootFile: "/dev/nvmen12p11",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			rootFile := getRootBlockFile(tt.devName)
			if rootFile != tt.rootFile {
				t1.Errorf("Test case name %s: Expected root file = (%s) got: %s", tt.name, tt.rootFile, rootFile)
			}
		})
	}

}

func TestSplitDevAndPartNum(t1 *testing.T) {

	testCases := []struct {
		name     string
		inputStr string
		devName  string
		partNum  int
	}{
		{
			name:     "test1",
			inputStr: "/var/lib/direct-csi/devices/xvdb-part-11",
			devName:  "/var/lib/direct-csi/devices/xvdb",
			partNum:  11,
		},
		{
			name:     "test2",
			inputStr: "/var/lib/direct-csi/devices/xvdb",
			devName:  "/var/lib/direct-csi/devices/xvdb",
			partNum:  0,
		},
		{
			name:     "test3",
			inputStr: "/var/lib/direct-csi/devices/nvmen1p-part-13",
			devName:  "/var/lib/direct-csi/devices/nvmen1p",
			partNum:  13,
		},
		{
			name:     "test4",
			inputStr: "/dev/sdb1",
			devName:  "/dev/sdb",
			partNum:  1,
		},
		{
			name:     "test5",
			inputStr: "/dev/sdb1",
			devName:  "/dev/sdb",
			partNum:  1,
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			dName, partNum := splitDevAndPartNum(tt.inputStr)
			if dName != tt.devName || partNum != tt.partNum {
				t1.Errorf("Test case name %s: Expected (devName, partNum) = (%s, %d) got: (%s, %d)", tt.name, tt.devName, tt.partNum, dName, partNum)
			}
		})
	}

}
