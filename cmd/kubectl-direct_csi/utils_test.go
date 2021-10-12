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

package main

import (
	"reflect"
	"testing"
)

func TestExpandSelector(t1 *testing.T) {
	testCases := []struct {
		selectors    []string
		expandedList []string
		expectErr    bool
	}{
		{
			selectors: []string{"/dev/xvd{b...f}"},
			expandedList: []string{
				"/dev/xvdb",
				"/dev/xvdc",
				"/dev/xvdd",
				"/dev/xvde",
				"/dev/xvdf",
			},
		},
		{
			selectors: []string{"node-{1...3}"},
			expandedList: []string{
				"node-1",
				"node-2",
				"node-3",
			},
		},
		{
			selectors: []string{"/dev/xvd{b...c}", "/dev/xvd{e...h}"},
			expandedList: []string{
				"/dev/xvdb",
				"/dev/xvdc",
				"/dev/xvde",
				"/dev/xvdf",
				"/dev/xvdg",
				"/dev/xvdh",
			},
		},
		{
			selectors: []string{"node-{1...3}", "node-{7...10}"},
			expandedList: []string{
				"node-1",
				"node-2",
				"node-3",
				"node-7",
				"node-8",
				"node-9",
				"node-10",
			},
		},
		{
			selectors: []string{"/dev/nvmen{1...2}p{1...2}"},
			expandedList: []string{
				"/dev/nvmen1p1",
				"/dev/nvmen1p2",
				"/dev/nvmen2p1",
				"/dev/nvmen2p2",
			},
		},
		{
			selectors:    []string{"/dev/xvd[b-f]"},
			expandedList: nil,
		},
		{
			selectors:    []string{"node-*"},
			expandedList: nil,
		},
		{
			selectors:    []string{"/dev/xvd{b..c}"},
			expandedList: nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{b..}"},
			expandedList: nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{..c}"},
			expandedList: nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{...}"},
			expandedList: nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{b...c}", "/dev/xvd{e...}"},
			expandedList: nil,
			expectErr:    true,
		},
	}

	for i, testCase := range testCases {
		list, err := expandSelector(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t1.Fatalf("case %v: did not expect error but got: %v", i+1, err)
		}
		if !reflect.DeepEqual(list, testCase.expandedList) {
			t1.Errorf("case %v: Expected list = %v, got %v", i+1, testCase.expandedList, list)
		}
	}
}

func TestHasGlob(t1 *testing.T) {
	testCases := []struct {
		inputStr string
		isGlob   bool
	}{
		{
			inputStr: "/dev/xvd[b-c]",
			isGlob:   true,
		},
		{
			inputStr: "/dev/xvd*",
			isGlob:   true,
		},
		{
			inputStr: "/dev/node[1-3]",
			isGlob:   true,
		},
		{
			inputStr: "/dev/node1*",
			isGlob:   true,
		},
		{
			inputStr: "/dev/xvd{b...c}",
		},
		{
			inputStr: "/dev/node{1...3}",
		},
	}

	for i, testCase := range testCases {
		isGlob, err := hasGlob(testCase.inputStr)
		if err != nil {
			t1.Fatalf("case %v: failed to check for glob: %v", i+1, err)
		}
		if testCase.isGlob != isGlob {
			t1.Errorf("case %v: Expected result = %v, got %v", i+1, testCase.isGlob, isGlob)
		}
	}
}

func TestSetIfNil(t1 *testing.T) {
	testCases := []struct {
		sliceA []string
		sliceB []string
		result []string
	}{
		{
			sliceA: []string{"abc"},
			sliceB: []string{"def"},
			result: nil,
		},
		{
			sliceA: nil,
			sliceB: []string{"def"},
			result: []string{"def"},
		},
		{
			sliceA: nil,
			sliceB: nil,
			result: nil,
		},
	}

	for i, testCase := range testCases {
		result := setIfNil(testCase.sliceA, testCase.sliceB)
		if !reflect.DeepEqual(result, testCase.result) {
			t1.Errorf("case %v: Expected result = %v, got %v", i+1, testCase.result, result)
		}
	}
}
