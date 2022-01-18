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

import "testing"

func TestIsFATFSType(t *testing.T) {
	testCases := []struct {
		fsType         string
		expectedResult bool
	}{
		{"fat", true},
		{"vfat", true},
		{"fat12", true},
		{"fat16", true},
		{"fat32", true},
		{"xfs", false},
	}

	for i, testCase := range testCases {
		result := isFATFSType(testCase.fsType)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsSwapFSType(t *testing.T) {
	testCases := []struct {
		fsType         string
		expectedResult bool
	}{
		{"linux-swap", true},
		{"swap", true},
		{"xfs", false},
	}

	for i, testCase := range testCases {
		result := isSwapFSType(testCase.fsType)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestFSTypeEqual(t *testing.T) {
	testCases := []struct {
		fsType1        string
		fsType2        string
		expectedResult bool
	}{
		{"vfat", "vfat", true},
		{"vfat", "fat32", true},
		{"swap", "swap", true},
		{"linux-swap", "swap", true},
		{"swap", "xfs", false},
		{"xfs", "vfat", false},
	}

	for i, testCase := range testCases {
		result := FSTypeEqual(testCase.fsType1, testCase.fsType2)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}
