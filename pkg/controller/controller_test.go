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

package controller

import (
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSelectDriveByTopology(t1 *testing.T) {

	testDriveSet := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive1",
			},
			Status: directcsi.DirectCSIDriveStatus{
				Topology: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive2",
			},
			Status: directcsi.DirectCSIDriveStatus{
				Topology: map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive3",
			},
			Status: directcsi.DirectCSIDriveStatus{
				Topology: map[string]string{"node": "N3", "rack": "RK3", "zone": "Z3", "region": "R3"},
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
	testDriveSet := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive1",
			},
			Status: directcsi.DirectCSIDriveStatus{
				FreeCapacity: 5000,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive2",
			},
			Status: directcsi.DirectCSIDriveStatus{
				FreeCapacity: 1000,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive3",
			},
			Status: directcsi.DirectCSIDriveStatus{
				FreeCapacity: 7000,
			},
		},
	}
	testCases := []struct {
		name              string
		capacityRange     *csi.CapacityRange
		selectedDriveList []directcsi.DirectCSIDrive
	}{
		{
			name:          "test1",
			capacityRange: &csi.CapacityRange{RequiredBytes: 2000, LimitBytes: 6000},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 5000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 7000,
					},
				},
			},
		},
		{
			name:          "test2",
			capacityRange: &csi.CapacityRange{RequiredBytes: 0, LimitBytes: 0},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 5000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 7000,
					},
				},
			},
		},
		{
			name:          "test3",
			capacityRange: &csi.CapacityRange{RequiredBytes: 2000, LimitBytes: 0},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 5000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 7000,
					},
				},
			},
		},
		{
			name:              "test4",
			capacityRange:     &csi.CapacityRange{RequiredBytes: 10000, LimitBytes: 0},
			selectedDriveList: []directcsi.DirectCSIDrive{},
		},
		{
			name:          "test5",
			capacityRange: &csi.CapacityRange{RequiredBytes: 500, LimitBytes: 800},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 5000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 7000,
					},
				},
			},
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
	testDriveSet := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive1",
			},
			Status: directcsi.DirectCSIDriveStatus{

				Filesystem: "ext4",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive2",
			},
			Status: directcsi.DirectCSIDriveStatus{
				Filesystem: "ext4",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive3",
			},
			Status: directcsi.DirectCSIDriveStatus{
				Filesystem: "xfs",
			},
		},
	}
	testCases := []struct {
		name              string
		fsType            string
		selectedDriveList []directcsi.DirectCSIDrive
	}{
		{
			name:   "test1",
			fsType: "ext4",
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "ext4",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "ext4",
					},
				},
			},
		},
		{
			name:   "test2",
			fsType: "xfs",
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "xfs",
					},
				},
			},
		},
		{
			name:   "test3",
			fsType: "",
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "ext4",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "ext4",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Filesystem: "xfs",
					},
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

func TestFilterDrivesByRequestedFormat(t1 *testing.T) {
	testCases := []struct {
		name              string
		driveList         []directcsi.DirectCSIDrive
		selectedDriveList []directcsi.DirectCSIDrive
	}{
		{
			name: "test1",
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusAvailable,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusInUse,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusInUse,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
			},
		},
		{
			name: "test2",
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
			},
		},
		{
			name: "test3",
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusTerminating,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusUnavailable,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						DriveStatus: directcsi.DriveStatusReady,
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			driveList := FilterDrivesByRequestFormat(tt.driveList)
			if !reflect.DeepEqual(driveList, tt.selectedDriveList) {
				t1.Errorf("Test case name %s: Expected drive list = %v, got %v", tt.name, tt.selectedDriveList, driveList)
			}
		})
	}
}
