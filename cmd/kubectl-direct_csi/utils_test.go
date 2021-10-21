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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
)

func TestGetValidSelectors(t *testing.T) {
	testCases := []struct {
		selectors    []string
		globs        []string
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
			globs: nil,
		},
		{
			selectors: []string{"node-{1...3}"},
			expandedList: []string{
				"node-1",
				"node-2",
				"node-3",
			},
			globs: nil,
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
			globs: nil,
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
			globs: nil,
		},
		{
			selectors: []string{"/dev/nvmen{1...2}p{1...2}"},
			expandedList: []string{
				"/dev/nvmen1p1",
				"/dev/nvmen1p2",
				"/dev/nvmen2p1",
				"/dev/nvmen2p2",
			},
			globs: nil,
		},
		{
			selectors:    []string{"/dev/xvd[b-f]"},
			globs:        []string{"/dev/xvd[b-f]"},
			expandedList: nil,
		},
		{
			selectors:    []string{"node-[1-3]"},
			globs:        []string{"node-[1-3]"},
			expandedList: nil,
		},
		{
			selectors:    []string{"/dev/xvd[b-c]", "/dev/xvd[e-h]"},
			globs:        []string{"/dev/xvd[b-c]", "/dev/xvd[e-h]"},
			expandedList: nil,
		},
		{
			selectors:    []string{"node-[1-3]", "node-[7-10]"},
			globs:        []string{"node-[1-3]", "node-[7-10]"},
			expandedList: nil,
		},
		{
			selectors:    []string{"/dev/nvmen*"},
			globs:        []string{"/dev/nvmen*"},
			expandedList: nil,
		},
		{
			selectors:    []string{"/dev/xvd{b..c}"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{b..}"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{..c}"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{...}"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{b...c}", "/dev/xvd{e...}"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
		{
			selectors:    []string{"/dev/xvd{b...c}", "/dev/xvd[e-f]"},
			expandedList: nil,
			globs:        nil,
			expectErr:    true,
		},
	}

	for i, testCase := range testCases {
		globs, expandedList, err := getValidSelectors(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t.Errorf("case %v: Expected err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(globs, testCase.globs) {
			t.Errorf("case %v: Expected globs = %v, got %v", i+1, testCase.globs, globs)
		}
		if !reflect.DeepEqual(expandedList, testCase.expandedList) {
			t.Errorf("case %v: Expected expandedList = %v, got %v", i+1, testCase.expandedList, expandedList)
		}
	}
}

func TestGetValidAccessTierSelectors(t *testing.T) {
	testCases := []struct {
		accessTiers []string
		result      []string
		expectErr   bool
	}{
		{
			accessTiers: []string{"hot", "cold"},
			result:      []string{"Hot", "Cold"},
		},
		{
			accessTiers: nil,
			result:      nil,
		},
		{
			accessTiers: []string{"ho", "cod"},
			result:      nil,
			expectErr:   true,
		},
	}

	for i, testCase := range testCases {
		result, err := getValidAccessTierSelectors(testCase.accessTiers)
		if err != nil && !testCase.expectErr {
			t.Errorf("case %v: Expected err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(result, testCase.result) {
			t.Errorf("case %v: Expected result = %v, got %v", i+1, testCase.result, result)
		}
	}
}

func TestGetValidDriveStatusSelectors(t *testing.T) {
	testCases := []struct {
		selectors  []string
		globs      []string
		statusList []directcsi.DriveStatus
		expectErr  bool
	}{
		{
			selectors:  []string{"available", "ready"},
			globs:      nil,
			statusList: []directcsi.DriveStatus{directcsi.DriveStatusAvailable, directcsi.DriveStatusReady},
		},
		{
			selectors:  []string{"available"},
			globs:      nil,
			statusList: []directcsi.DriveStatus{directcsi.DriveStatusAvailable},
		},
		{
			selectors:  []string{"*avail*"},
			globs:      []string{"*avail*"},
			statusList: nil,
		},
		{
			selectors:  []string{"xyz"},
			globs:      nil,
			statusList: nil,
			expectErr:  true,
		},
	}

	for i, testCase := range testCases {
		globs, statusList, err := getValidDriveStatusSelectors(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t.Errorf("case %v: Expected err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(globs, testCase.globs) {
			t.Errorf("case %v: Expected globs = %v, got %v", i+1, testCase.globs, globs)
		}
		if !reflect.DeepEqual(statusList, testCase.statusList) {
			t.Errorf("case %v: Expected statusList = %v, got %v", i+1, testCase.statusList, statusList)
		}
	}
}

func TestGetValidVolumeStatusSelectors(t *testing.T) {
	testCases := []struct {
		selectors  []string
		statusList []string
		expectErr  bool
	}{
		{
			selectors:  []string{"published"},
			statusList: []string{"published"},
		},
		{
			selectors:  []string{"staged"},
			statusList: []string{"staged"},
		},
		{
			selectors:  []string{"staged", "published"},
			statusList: []string{"staged", "published"},
		},
		{
			selectors:  []string{"sta", "zyx"},
			statusList: nil,
			expectErr:  true,
		},
	}

	for i, testCase := range testCases {
		statusList, err := getValidVolumeStatusSelectors(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t.Errorf("case %v: Expected err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(statusList, testCase.statusList) {
			t.Errorf("case %v: Expected statusList = %v, got %v", i+1, testCase.statusList, statusList)
		}
	}
}
