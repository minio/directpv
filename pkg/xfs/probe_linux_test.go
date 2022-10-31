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

package xfs

import (
	"os"
	"testing"
)

func TestReadSuperBlock(t *testing.T) {
	testCases := []struct {
		filename      string
		fsuuid        string
		label         string
		totalCapacity uint64
		freeCapacity  uint64
		expectErr     bool
	}{
		{"xfs.testdata", "2dc39938-8a84-4078-abec-159bfae4aa0f", "", 52428800, 46743552, false},
		{"zero.testdata", "", "", 0, 0, true},
		{"empty.testdata", "", "", 0, 0, true},
	}

	for i, testCase := range testCases {
		func() {
			file, err := os.Open(testCase.filename)
			if err != nil {
				t.Fatalf("case %v: %v", i+1, err)
			}
			defer file.Close()

			fsuuid, label, totalCapacity, freeCapacity, err := readSuperBlock(file)
			if testCase.expectErr {
				if err == nil {
					t.Fatalf("case %v: expected error, but succeeded", i+1)
				}
				return
			}

			if err != nil {
				t.Fatalf("case %v: %v", i+1, err)
			}

			if fsuuid != testCase.fsuuid {
				t.Fatalf("case %v: FSUUID: expected: %v, got: %v", i+1, testCase.fsuuid, fsuuid)
			}

			if label != testCase.label {
				t.Fatalf("case %v: label: expected: %v, got: %v", i+1, testCase.label, label)
			}

			if totalCapacity != testCase.totalCapacity {
				t.Fatalf("case %v: totalCapacity: expected: %v, got: %v", i+1, testCase.totalCapacity, totalCapacity)
			}

			if freeCapacity != testCase.freeCapacity {
				t.Fatalf("case %v: freeCapacity: expected: %v, got: %v", i+1, testCase.freeCapacity, freeCapacity)
			}
		}()
	}
}
