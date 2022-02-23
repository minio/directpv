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

package installer

import (
	"fmt"
	"testing"
)

func TestTrimMinorVersion(t *testing.T) {
	testCases := []struct {
		minor          string
		expectedResult string
	}{
		{"18", "18"},
		{"18+", "18"},
		{"18-", "18"},
		{"18incompat", "18"},
		{"0-incompat", "0"},
	}

	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("case-%v", i), func(t *testing.T) {
			t.Parallel()
			result, err := trimMinorVersion(testCase.minor)
			if err != nil {
				t.Fatalf("unexpected error; %v", err)
			}
			if result != testCase.expectedResult {
				t.Fatalf("expected: %v, got: %v", testCase.expectedResult, result)
			}
		})
	}
}

func TestTrimMinorVersionError(t *testing.T) {
	testCases := []struct {
		minor string
	}{
		{"incompat"},
		{"-2"},
	}

	for i, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("case-%v", i), func(t *testing.T) {
			t.Parallel()
			_, err := trimMinorVersion(testCase.minor)
			if err == nil {
				t.Fatalf("expected error; but succeeded for %v", testCase.minor)
			}
		})
	}
}
