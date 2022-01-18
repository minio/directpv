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

package ext4

import (
	"os"
	"testing"
)

func TestProbe(t *testing.T) {
	testCases := []struct {
		filename      string
		id            string
		fsType        string
		totalCapacity uint64
		freeCapacity  uint64
		expectErr     bool
	}{
		{"ext4.testdata", "caa7c38c-d58f-4b5c-9f1d-f551a39ed095", "ext4", 52428800, 45506560, false},
		{"ext3.testdata", "c0839b43-3022-4ad2-ad3e-00e0dae1a653", "ext4", 67108864, 59452416, false},
		{"ext2.testdata1", "654c8a21-8afa-4ae8-9305-7c436015f27a", "ext4", 67108864, 63664128, false},
		{"ext2.testdata2", "00000000-0000-0000-0000-000000000000", "ext4", 67108864, 63664128, false},
		{"ext2.testdata3", "00000000-0000-0000-0000-000000000000", "ext4", 67108864, 64964608, false},
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

			sb, err := Probe(file)
			if testCase.expectErr {
				if err == nil {
					t.Fatalf("case %v: expected error, but succeeded", i+1)
				}
				return
			}

			if err != nil {
				t.Fatalf("case %v: %v", i+1, err)
			}

			if sb.ID() != testCase.id {
				t.Fatalf("case %v: ID: expected: %v, got: %v", i+1, testCase.id, sb.ID())
			}

			if sb.Type() != testCase.fsType {
				t.Fatalf("case %v: Type: expected: %v, got: %v", i+1, testCase.fsType, sb.Type())
			}

			if sb.TotalCapacity() != testCase.totalCapacity {
				t.Fatalf("case %v: TotalCapacity: expected: %v, got: %v", i+1, testCase.totalCapacity, sb.TotalCapacity())
			}

			if sb.FreeCapacity() != testCase.freeCapacity {
				t.Fatalf("case %v: FreeCapacity: expected: %v, got: %v", i+1, testCase.freeCapacity, sb.FreeCapacity())
			}
		}()
	}
}
