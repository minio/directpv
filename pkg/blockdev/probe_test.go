/*
 * This file is part of MinIO Direct CSI
 * Copyright (c) 2021 MinIO, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package blockdev

import (
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/minio/directpv/pkg/blockdev/gpt"
	"github.com/minio/directpv/pkg/blockdev/mbr"
	"github.com/minio/directpv/pkg/blockdev/parttable"
)

type testPartTable struct {
	uuid       string
	partType   string
	partitions map[int]*parttable.Partition
}

func (tpt *testPartTable) equal(pt parttable.PartTable) bool {
	if pt == nil {
		return false
	}

	if tpt.uuid != pt.UUID() {
		return false
	}

	if tpt.partType != pt.Type() {
		return false
	}

	return reflect.DeepEqual(tpt.partitions, pt.Partitions())
}

func TestMBRProbe(t *testing.T) {
	testCase1Result := &testPartTable{"", "msdos", map[int]*parttable.Partition{}}
	testCase2Result := &testPartTable{
		"",
		"msdos",
		map[int]*parttable.Partition{
			1: {Number: 1, Type: parttable.Primary},
			2: {Number: 2, Type: parttable.Extended},
			5: {Number: 5, Type: parttable.Logical},
			6: {Number: 6, Type: parttable.Logical},
		},
	}
	testCase3Result := &testPartTable{
		"",
		"msdos",
		map[int]*parttable.Partition{
			1: {Number: 1, Type: parttable.Primary},
			2: {Number: 2, Type: parttable.Primary},
			3: {Number: 3, Type: parttable.Primary},
			4: {Number: 4, Type: parttable.Primary},
		},
	}

	testCases := []struct {
		testDataFile string
		result       *testPartTable
		err          error
	}{
		{"msdos.empty-parts.testdata", testCase1Result, nil},
		{"msdos.logical-partitions.testdata", testCase2Result, nil},
		{"msdos.only-primary-partitions.testdata", testCase3Result, nil},
		{"gpt.testdata", nil, mbr.ErrGPTProtectiveMBR},
		{"zero.testdata", nil, parttable.ErrPartTableNotFound},
	}

	for i, testCase := range testCases {
		devFile, err := os.Open(testCase.testDataFile)
		if err != nil {
			t.Fatalf("case %v: %v: %v", i+1, testCase.testDataFile, err)
		}
		defer devFile.Close()

		result, err := mbr.Probe(devFile)
		if !errors.Is(err, testCase.err) {
			t.Fatalf("case %v: err: expected: %v, got: %v", i+1, testCase.err, err)
		}
		if testCase.result != nil {
			if !testCase.result.equal(result) {
				t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.result, result)
			}
		} else if result != nil {
			t.Fatalf("case %v: result: expected: <nil>, got: %v", i+1, result)
		}
	}
}

func TestGPTProbe(t *testing.T) {
	testCase1Result := &testPartTable{
		"6ce102c7-cfc2-4b1c-b658-02ba8cd9f58f",
		"gpt",
		map[int]*parttable.Partition{
			4: {Number: 4, UUID: "8a7d885f-88ba-4734-bbc7-90881480a5a6", Type: parttable.Primary},
			1: {Number: 1, UUID: "0d167e49-2c8d-4c6c-ad82-b5e66b6a9eda", Type: parttable.Primary},
			2: {Number: 2, UUID: "a183b96b-072c-4236-ae9a-d8adce39859d", Type: parttable.Primary},
			3: {Number: 3, UUID: "89fc4f86-1519-47c8-a9f1-11ed504c8f18", Type: parttable.Primary},
		},
	}

	testCases := []struct {
		testDataFile string
		result       *testPartTable
		err          error
	}{
		{"gpt.testdata", testCase1Result, nil},
		{"msdos.empty-parts.testdata", nil, io.EOF},
		{"msdos.logical-partitions.testdata", nil, parttable.ErrPartTableNotFound},
		{"msdos.only-primary-partitions.testdata", nil, io.EOF},
		{"zero.testdata", nil, parttable.ErrPartTableNotFound},
	}

	for i, testCase := range testCases {
		devFile, err := os.Open(testCase.testDataFile)
		if err != nil {
			t.Fatalf("case %v: %v: %v", i+1, testCase.testDataFile, err)
		}
		defer devFile.Close()

		if _, err = devFile.Seek(512, io.SeekStart); err != nil {
			t.Fatalf("case %v: %v: %v", i+1, testCase.testDataFile, err)
		}

		result, err := gpt.Probe(devFile)

		if !errors.Is(err, testCase.err) {
			t.Fatalf("case %v: err: expected: %v, got: %v", i+1, testCase.err, err)
		}
		if testCase.result != nil {
			if !testCase.result.equal(result) {
				t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.result, result)
			}
		} else if result != nil {
			t.Fatalf("case %v: result: expected: <nil>, got: %v", i+1, result)
		}
	}
}
