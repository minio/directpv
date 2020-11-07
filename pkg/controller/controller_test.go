// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package controller

import (
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
)

func TestSelectDriveByTopology(t1 *testing.T) {

	testDriveSet := []direct_csi.DirectCSIDrive{
		{
			Name: "drive1",
			Topology: &csi.Topology{
				Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
			},
		},
		{
			Name: "drive2",
			Topology: &csi.Topology{
				Segments: map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"},
			},
		},
		{
			Name: "drive3",
			Topology: &csi.Topology{
				Segments: map[string]string{"node": "N3", "rack": "RK3", "zone": "Z3", "region": "R3"},
			},
		},
	}

	testCases := []struct {
		name              string
		topologyRequest   *csi.Topology
		errExpected       bool
		selectedDriveName string
	}{
		{
			name:              "test1",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"}},
			errExpected:       false,
			selectedDriveName: "drive2",
		},
		{
			name:              "test2",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N3", "rack": "RK3", "zone": "Z3", "region": "R3"}},
			errExpected:       false,
			selectedDriveName: "drive3",
		},
		{
			name:              "test3",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N4", "rack": "RK2", "zone": "Z4", "region": "R2"}},
			errExpected:       true,
			selectedDriveName: "",
		},
		{
			name:              "test4",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N3", "rack": "RK3"}},
			errExpected:       false,
			selectedDriveName: "drive3",
		},
		{
			name:              "test5",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N1", "rack": "RK5"}},
			errExpected:       true,
			selectedDriveName: "",
		},
		{
			name:              "test5",
			topologyRequest:   &csi.Topology{Segments: map[string]string{"node": "N1"}},
			errExpected:       false,
			selectedDriveName: "drive1",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			selectedDrive, err := selectDriveByTopology(tt.topologyRequest, testDriveSet)
			if tt.errExpected && err == nil {
				t1.Fatalf("Test case name %s: Expected error but succeeded", tt.name)
			} else if selectedDrive.Name != tt.selectedDriveName {
				t1.Errorf("Test case name %s: Expected drive name = %s, got %v", tt.name, tt.selectedDriveName, selectedDrive.Name)
			}
		})
	}
}

func TestFilterDrivesByCapacityRange(t1 *testing.T) {
	testDriveSet := []direct_csi.DirectCSIDrive{
		{
			Name:         "drive1",
			FreeCapacity: 5000,
		},
		{
			Name:         "drive2",
			FreeCapacity: 1000,
		},
		{
			Name:         "drive3",
			FreeCapacity: 7000,
		},
	}
	testCases := []struct {
		name              string
		capacityRange     *csi.CapacityRange
		selectedDriveList []direct_csi.DirectCSIDrive
	}{
		{
			name:          "test1",
			capacityRange: &csi.CapacityRange{RequiredBytes: 2000, LimitBytes: 6000},
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:         "drive1",
					FreeCapacity: 5000,
				},
			},
		},
		{
			name:          "test2",
			capacityRange: &csi.CapacityRange{RequiredBytes: 0, LimitBytes: 0},
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:         "drive1",
					FreeCapacity: 5000,
				},
				{
					Name:         "drive2",
					FreeCapacity: 1000,
				},
				{
					Name:         "drive3",
					FreeCapacity: 7000,
				},
			},
		},
		{
			name:          "test3",
			capacityRange: &csi.CapacityRange{RequiredBytes: 2000, LimitBytes: 0},
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:         "drive1",
					FreeCapacity: 5000,
				},
				{
					Name:         "drive3",
					FreeCapacity: 7000,
				},
			},
		},
		{
			name:              "test4",
			capacityRange:     &csi.CapacityRange{RequiredBytes: 10000, LimitBytes: 0},
			selectedDriveList: []direct_csi.DirectCSIDrive{},
		},
		{
			name:              "test5",
			capacityRange:     &csi.CapacityRange{RequiredBytes: 500, LimitBytes: 800},
			selectedDriveList: []direct_csi.DirectCSIDrive{},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			driveList := FilterDrivesByCapacityRange(tt.capacityRange, testDriveSet)
			if !reflect.DeepEqual(driveList, tt.selectedDriveList) {
				t1.Errorf("Test case name %s: Expected drive list = %v, got %v", tt.name, tt.selectedDriveList, driveList)
			}
		})
	}
}

func TestFilterDrivesByFsType(t1 *testing.T) {
	testDriveSet := []direct_csi.DirectCSIDrive{
		{
			Name:       "drive1",
			Filesystem: "ext4",
		},
		{
			Name:       "drive2",
			Filesystem: "ext4",
		},
		{
			Name:       "drive3",
			Filesystem: "xfs",
		},
	}
	testCases := []struct {
		name              string
		fsType            string
		selectedDriveList []direct_csi.DirectCSIDrive
	}{
		{
			name:   "test1",
			fsType: "ext4",
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:       "drive1",
					Filesystem: "ext4",
				},
				{
					Name:       "drive2",
					Filesystem: "ext4",
				},
			},
		},
		{
			name:   "test2",
			fsType: "xfs",
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:       "drive3",
					Filesystem: "xfs",
				},
			},
		},
		{
			name:   "test3",
			fsType: "",
			selectedDriveList: []direct_csi.DirectCSIDrive{
				{
					Name:       "drive1",
					Filesystem: "ext4",
				},
				{
					Name:       "drive2",
					Filesystem: "ext4",
				},
				{
					Name:       "drive3",
					Filesystem: "xfs",
				},
			},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			driveList := FilterDrivesByFsType(tt.fsType, testDriveSet)
			if !reflect.DeepEqual(driveList, tt.selectedDriveList) {
				t1.Errorf("Test case name %s: Expected drive list = %v, got %v", tt.name, tt.selectedDriveList, driveList)
			}
		})
	}
}
