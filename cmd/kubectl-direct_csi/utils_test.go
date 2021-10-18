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

func TestExpandSelectors(t1 *testing.T) {
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
		list, err := expandSelectors(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t1.Fatalf("case %v: did not expect error but got: %v", i+1, err)
		}
		if !reflect.DeepEqual(list, testCase.expandedList) {
			t1.Errorf("case %v: Expected list = %v, got %v", i+1, testCase.expandedList, list)
		}
	}
}

func TestSplitSelectors(t1 *testing.T) {
	testCases := []struct {
		selectors         []string
		globSelectors     []string
		ellipsesSelectors []string
	}{
		{
			selectors:         []string{"/dev/xvd{a...c}", "/dev/xvd{e...f}"},
			globSelectors:     nil,
			ellipsesSelectors: []string{"/dev/xvd{a...c}", "/dev/xvd{e...f}"},
		},
		{
			selectors:         []string{"/dev/xvd[a-c]", "/dev/xvd[e-f]"},
			globSelectors:     []string{"/dev/xvd[a-c]", "/dev/xvd[e-f]"},
			ellipsesSelectors: nil,
		},
		{
			selectors:         []string{"/dev/xvd[a-c]"},
			globSelectors:     []string{"/dev/xvd[a-c]"},
			ellipsesSelectors: nil,
		},
		{
			selectors:         []string{"/dev/xvd{a...c}", "/dev/xvd[e-f]", "/dev/xvd{g...h}", "/dev/xvd[i-z]"},
			globSelectors:     []string{"/dev/xvd[e-f]", "/dev/xvd[i-z]"},
			ellipsesSelectors: []string{"/dev/xvd{a...c}", "/dev/xvd{g...h}"},
		},
	}
	for i, testCase := range testCases {
		globSelectors, ellipsesSelectors := splitSelectors(testCase.selectors)
		if !reflect.DeepEqual(globSelectors, testCase.globSelectors) {
			t1.Errorf("case %v: Expected globSelectorList = %v, got %v", i+1, testCase.globSelectors, globSelectors)
		}
		if !reflect.DeepEqual(ellipsesSelectors, testCase.ellipsesSelectors) {
			t1.Errorf("case %v: Expected ellipsesSelectorList = %v, got %v", i+1, testCase.ellipsesSelectors, ellipsesSelectors)
		}
	}
}
