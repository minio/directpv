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

package main

import (
	"reflect"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"
)

func TestGetValidSelectors(t *testing.T) {
	testCases := []struct {
		selectors []string
		globs     []string
		values    []utils.LabelValue
		expectErr bool
	}{
		{
			selectors: []string{"xvd{b...f}"},
			values: []utils.LabelValue{
				utils.LabelValue("xvdb"),
				utils.LabelValue("xvdc"),
				utils.LabelValue("xvdd"),
				utils.LabelValue("xvde"),
				utils.LabelValue("xvdf"),
			},
			globs: nil,
		},
		{
			selectors: []string{"node-{1...3}"},
			values: []utils.LabelValue{
				utils.LabelValue("node-1"),
				utils.LabelValue("node-2"),
				utils.LabelValue("node-3"),
			},
			globs: nil,
		},
		{
			selectors: []string{"xvd{b...c}", "xvd{e...h}"},
			values: []utils.LabelValue{
				utils.LabelValue("xvdb"),
				utils.LabelValue("xvdc"),
				utils.LabelValue("xvde"),
				utils.LabelValue("xvdf"),
				utils.LabelValue("xvdg"),
				utils.LabelValue("xvdh"),
			},
			globs: nil,
		},
		{
			selectors: []string{"node-{1...3}", "node-{7...10}"},
			values: []utils.LabelValue{
				utils.LabelValue("node-1"),
				utils.LabelValue("node-2"),
				utils.LabelValue("node-3"),
				utils.LabelValue("node-7"),
				utils.LabelValue("node-8"),
				utils.LabelValue("node-9"),
				utils.LabelValue("node-10"),
			},
			globs: nil,
		},
		{
			selectors: []string{"nvmen{1...2}p{1...2}"},
			values: []utils.LabelValue{
				utils.LabelValue("nvmen1p1"),
				utils.LabelValue("nvmen1p2"),
				utils.LabelValue("nvmen2p1"),
				utils.LabelValue("nvmen2p2"),
			},
			globs: nil,
		},
		{
			selectors: []string{"xvd[b-f]"},
			globs:     []string{"xvd[b-f]"},
			values:    nil,
		},
		{
			selectors: []string{"node-[1-3]"},
			globs:     []string{"node-[1-3]"},
			values:    nil,
		},
		{
			selectors: []string{"xvd[b-c]", "xvd[e-h]"},
			globs:     []string{"xvd[b-c]", "xvd[e-h]"},
			values:    nil,
		},
		{
			selectors: []string{"node-[1-3]", "node-[7-10]"},
			globs:     []string{"node-[1-3]", "node-[7-10]"},
			values:    nil,
		},
		{
			selectors: []string{"nvmen*"},
			globs:     []string{"nvmen*"},
			values:    nil,
		},
		{
			selectors: []string{"xvd{b..c}"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
		{
			selectors: []string{"xvd{b..}"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
		{
			selectors: []string{"xvd{..c}"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
		{
			selectors: []string{"xvd{...}"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
		{
			selectors: []string{"xvd{b...c}", "xvd{e...}"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
		{
			selectors: []string{"xvd{b...c}", "xvd[e-f]"},
			values:    nil,
			globs:     nil,
			expectErr: true,
		},
	}

	for i, testCase := range testCases {
		globs, values, err := getValidSelectors(testCase.selectors)
		if err != nil && !testCase.expectErr {
			t.Errorf("case %v: Expected err to nil, got %v", i+1, err)
		}
		if !reflect.DeepEqual(globs, testCase.globs) {
			t.Errorf("case %v: Expected globs = %v, got %v", i+1, testCase.globs, globs)
		}
		if !reflect.DeepEqual(values, testCase.values) {
			t.Errorf("case %v: Expected expandedList = %v, got %v", i+1, testCase.values, values)
		}
	}
}

func TestGetValidAccessTierSelectors(t *testing.T) {
	testCases := []struct {
		accessTiers []string
		result      []utils.LabelValue
		expectErr   bool
	}{
		{
			accessTiers: []string{"hot", "cold"},
			result:      []utils.LabelValue{utils.LabelValue("Hot"), utils.LabelValue("Cold")},
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
