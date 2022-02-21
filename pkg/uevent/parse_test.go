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

package uevent

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	header := append([]byte(libudev), 0xfe, 0xed, 0xca, 0xfe, 0, 0, 0, 0)

	case3Msg := append(header, 16)
	case4Msg := append(header, 18)
	case5Msg := append(header, 17)
	case6Msg := append(header, 17, 'a')
	case7Msg := append(header, 17, 'a', '=', '1')
	case8Msg := append(header, 17, 0, 'a', 0, 'b', '=', '1', 0)

	testCases := []struct {
		msg            []byte
		expectedResult map[string]string
		expectErr      bool
	}{
		{nil, nil, true},
		{[]byte(libudev + "1234"), nil, true},
		{case3Msg, nil, true},
		{case4Msg, nil, true},
		{case5Msg, map[string]string{}, false},
		{case6Msg, map[string]string{"a": ""}, false},
		{case7Msg, map[string]string{"a": "1"}, false},
		{case8Msg, map[string]string{"a": "", "b": "1"}, false},
	}

	for i, testCase := range testCases {
		result, err := parse(testCase.msg)
		if testCase.expectErr {
			if err == nil {
				t.Fatalf("case %v: expected error; but succeeded", i+1)
			}
			continue
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %+v; got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}
