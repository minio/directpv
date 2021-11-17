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
	"github.com/minio/direct-csi/pkg/client"
)

func TestGetValidSelectors(t *testing.T) {
	testCases := []struct {
		selectors []string
		globs     []string
		values    []client.LabelValue
		expectErr bool
	}{
		{
			selectors: []string{"xvd{b...f}"},
			values: []client.LabelValue{
				client.LabelValue("xvdb"),
				client.LabelValue("xvdc"),
				client.LabelValue("xvdd"),
				client.LabelValue("xvde"),
				client.LabelValue("xvdf"),
			},
			globs: nil,
		},
		{
			selectors: []string{"node-{1...3}"},
			values: []client.LabelValue{
				client.LabelValue("node-1"),
				client.LabelValue("node-2"),
				client.LabelValue("node-3"),
			},
			globs: nil,
		},
		{
			selectors: []string{"xvd{b...c}", "xvd{e...h}"},
			values: []client.LabelValue{
				client.LabelValue("xvdb"),
				client.LabelValue("xvdc"),
				client.LabelValue("xvde"),
				client.LabelValue("xvdf"),
				client.LabelValue("xvdg"),
				client.LabelValue("xvdh"),
			},
			globs: nil,
		},
		{
			selectors: []string{"node-{1...3}", "node-{7...10}"},
			values: []client.LabelValue{
				client.LabelValue("node-1"),
				client.LabelValue("node-2"),
				client.LabelValue("node-3"),
				client.LabelValue("node-7"),
				client.LabelValue("node-8"),
				client.LabelValue("node-9"),
				client.LabelValue("node-10"),
			},
			globs: nil,
		},
		{
			selectors: []string{"nvmen{1...2}p{1...2}"},
			values: []client.LabelValue{
				client.LabelValue("nvmen1p1"),
				client.LabelValue("nvmen1p2"),
				client.LabelValue("nvmen2p1"),
				client.LabelValue("nvmen2p2"),
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
		result      []client.LabelValue
		expectErr   bool
	}{
		{
			accessTiers: []string{"hot", "cold"},
			result:      []client.LabelValue{client.LabelValue("Hot"), client.LabelValue("Cold")},
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
