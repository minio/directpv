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
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	KB = 1 << 10
	MB = KB << 10

	mb50  = 50 * MB
	mb100 = 100 * MB
	mb20  = 20 * MB
	mb30  = 30 * MB
)

func TestSelectDrivesByTopology(t1 *testing.T) {

	getDriveNameSet := func(drives []directcsi.DirectCSIDrive) []string {
		driveNames := []string{}
		for _, drive := range drives {
			driveNames = append(driveNames, drive.Name)
		}
		return driveNames
	}

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
				Name: "drive11",
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
				Name: "drive22",
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
		name               string
		topologyRequest    *csi.Topology
		errExpected        bool
		selectedDriveNames []string
	}{
		{
			name:               "test1",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"}},
			errExpected:        false,
			selectedDriveNames: []string{"drive2", "drive22"},
		},
		{
			name:               "test2",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N3", "rack": "RK3", "zone": "Z3", "region": "R3"}},
			errExpected:        false,
			selectedDriveNames: []string{"drive3"},
		},
		{
			name:               "test3",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N4", "rack": "RK2", "zone": "Z4", "region": "R2"}},
			errExpected:        true,
			selectedDriveNames: []string{},
		},
		{
			name:               "test4",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N3", "rack": "RK3"}},
			errExpected:        false,
			selectedDriveNames: []string{"drive3"},
		},
		{
			name:               "test5",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N1", "rack": "RK5"}},
			errExpected:        true,
			selectedDriveNames: []string{},
		},
		{
			name:               "test5",
			topologyRequest:    &csi.Topology{Segments: map[string]string{"node": "N1"}},
			errExpected:        false,
			selectedDriveNames: []string{"drive1", "drive11"},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			selectedDrives, err := selectDrivesByTopology(tt.topologyRequest, testDriveSet)
			if tt.errExpected && err == nil {
				t1.Fatalf("Test case name %s: Expected error but succeeded", tt.name)
			} else if !reflect.DeepEqual(getDriveNameSet(selectedDrives), tt.selectedDriveNames) {
				t1.Errorf("Test case name %s: Expected drive names = %s, got %v", tt.name, tt.selectedDriveNames, getDriveNameSet(selectedDrives))
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

func TestFilterDrivesByParameters(t1 *testing.T) {
	testCases := []struct {
		name              string
		parameters        map[string]string
		driveList         []directcsi.DirectCSIDrive
		selectedDriveList []directcsi.DirectCSIDrive
		expectError       bool
	}{
		{
			name:       "test1",
			parameters: map[string]string{"abc": "def"},
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierCold,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierCold,
					},
				},
			},
			expectError: false,
		},
		{
			name:       "test2",
			parameters: map[string]string{"direct-csi-min-io/access-tier": "hot", "abc": "def"},
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
			},
			expectError: false,
		},
		{
			name:       "test3",
			parameters: map[string]string{"direct-csi-min-io/access-tier": "cold", "abc": "def"},
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{},
			expectError:       false,
		},
		{
			name:       "test4",
			parameters: map[string]string{"direct-csi-min-io/access-tier": "inVaLidValue", "abc": "def"},
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierHot,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						AccessTier: directcsi.AccessTierUnknown,
					},
				},
			},
			selectedDriveList: []directcsi.DirectCSIDrive{},
			expectError:       true,
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			driveList, err := FilterDrivesByParameters(tt.parameters, tt.driveList)
			if err != nil {
				if !tt.expectError {
					t1.Errorf("Test case name %s: Failed with %v", tt.name, err)
				}
			} else {
				if !reflect.DeepEqual(driveList, tt.selectedDriveList) {
					t1.Errorf("Test case name %s: Expected drive list = %v, got %v", tt.name, tt.selectedDriveList, driveList)
				}
			}
		})
	}
}

func createFakeController() *ControllerServer {
	return &ControllerServer{
		NodeID:          "test-node-1",
		Identity:        "test-identity-1",
		Rack:            "test-rack-1",
		Zone:            "test-zone-1",
		Region:          "test-region-1",
		directcsiClient: fakedirect.NewSimpleClientset(),
	}
}

func TestCreateAndDeleteVolumeRPCs(t *testing.T) {

	getTopologySegmentsForNode := func(node string) map[string]string {
		switch node {
		case "N1":
			return map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"}
		case "N2":
			return map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"}
		default:
			return map[string]string{}
		}
	}

	createTestDrive100MB := func(node, drive string) *directcsi.DirectCSIDrive {
		return &directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: drive,
				Finalizers: []string{
					string(directcsi.DirectCSIDriveFinalizerDataProtection),
				},
			},
			Status: directcsi.DirectCSIDriveStatus{
				NodeName:          node,
				Filesystem:        string(sys.FSTypeXFS),
				DriveStatus:       directcsi.DriveStatusReady,
				FreeCapacity:      mb100,
				AllocatedCapacity: int64(0),
				TotalCapacity:     mb100,
				Topology:          getTopologySegmentsForNode(node),
			},
		}
	}

	create20MBVolumeRequest := func(volName string, requestedNode string) csi.CreateVolumeRequest {
		return csi.CreateVolumeRequest{
			Name: volName,
			CapacityRange: &csi.CapacityRange{
				RequiredBytes: mb20,
			},
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{
							FsType: "xfs",
						},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
			},
			AccessibilityRequirements: &csi.TopologyRequirement{
				Preferred: []*csi.Topology{
					{
						Segments: getTopologySegmentsForNode(requestedNode),
					},
				},
				Requisite: []*csi.Topology{
					{
						Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
					},
					{
						Segments: map[string]string{"node": "N2", "rack": "RK2", "zone": "Z2", "region": "R2"},
					},
				},
			},
		}
	}

	createDeleteVolumeRequest := func(volName string) csi.DeleteVolumeRequest {
		return csi.DeleteVolumeRequest{
			VolumeId: volName,
		}
	}

	testDriveObjects := []runtime.Object{
		// Drives from Node N1
		createTestDrive100MB("N1", "D1"),
		createTestDrive100MB("N1", "D2"),
		// Drives from Node N2
		createTestDrive100MB("N2", "D3"),
		createTestDrive100MB("N2", "D4"),
	}

	createVolumeRequests := []csi.CreateVolumeRequest{
		// Volume requests for drives in Node N1
		create20MBVolumeRequest("volume-1", "N1"),
		create20MBVolumeRequest("volume-2", "N1"),
		// Volume requests for drives in Node N2
		create20MBVolumeRequest("volume-3", "N2"),
		create20MBVolumeRequest("volume-4", "N2"),
	}

	ctx := context.TODO()
	cl := createFakeController()
	cl.directcsiClient = fakedirect.NewSimpleClientset(testDriveObjects...)
	directCSIClient := cl.directcsiClient.DirectV1beta2()

	for _, cvReq := range createVolumeRequests {
		volName := cvReq.GetName()
		// Step 1: Call CreateVolume RPC
		cvRes, err := cl.CreateVolume(ctx, &cvReq)
		if err != nil {
			t.Errorf("[%s] Create volume failed: %v", volName, err)
		}
		// Step 2: Check if the response has the corresponding volume ID
		vol := cvRes.GetVolume()
		volumeID := vol.GetVolumeId()
		if volumeID != volName {
			t.Errorf("[%s] Wrong volumeID found in the response. Expected: %v, Got: %v", volName, volName, volumeID)
		}
		// Step 3: Check the the accessible topology in the response is matching with the request
		if !reflect.DeepEqual(vol.GetAccessibleTopology(),
			cvReq.GetAccessibilityRequirements().GetPreferred()) {
			t.Errorf("[%s] Accessible topology not matching with preferred topology in the request. Expected: %v, Got: %v", volName, cvReq.GetAccessibilityRequirements().GetPreferred(), vol.GetAccessibleTopology())
		}
		// Step 4: Fetch the created volume object by volumeID
		volObj, gErr := directCSIClient.DirectCSIVolumes().Get(ctx, volumeID, metav1.GetOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		})
		if gErr != nil {
			t.Fatalf("[%s] Volume (%s) not found. Error: %v", volName, volumeID, gErr)
		}
		// Step 5: Check if the finalizers were set correctly
		volFinalizers := volObj.GetFinalizers()
		for _, f := range volFinalizers {
			switch f {
			case directcsi.DirectCSIVolumeFinalizerPVProtection:
			case directcsi.DirectCSIVolumeFinalizerPurgeProtection:
			default:
				t.Errorf("[%s] Unknown finalizer found: %v", volName, f)
			}
		}
		// Step 6: Check if the total capacity is set correctly
		if volObj.Status.TotalCapacity != cvReq.CapacityRange.RequiredBytes {
			t.Errorf("[%s] Expected total capacity of the volume to be %d but got %d", volName, cvReq.CapacityRange.RequiredBytes, volObj.Status.TotalCapacity)
		}
	}

	// Fetch the drive objects
	driveList, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if err != nil {
		t.Errorf("Listing drives failed: %v", err)
	}

	// Checks to ensure if the volumes were equally distributed among drives
	// And also check if the drive status were updated properly
	for _, drive := range driveList.Items {
		if len(drive.GetFinalizers()) > 2 {
			t.Errorf("Volumes were not equally distributed among drives. Drive name: %v Finaliers: %v", drive.Name, drive.GetFinalizers())
		}
		if drive.Status.DriveStatus != directcsi.DriveStatusInUse {
			t.Errorf("Expected drive(%s) status: %s, But got: %v", drive.Name, string(directcsi.DriveStatusInUse), string(drive.Status.DriveStatus))
		}
		if drive.Status.FreeCapacity != (mb100 - mb20) {
			t.Errorf("Expected drive(%s) FreeCapacity: %d, But got: %d", drive.Name, (mb100 - mb20), drive.Status.FreeCapacity)
		}
		if drive.Status.AllocatedCapacity != mb20 {
			t.Errorf("Expected drive(%s) AllocatedCapacity: %d, But got: %d", drive.Name, mb20, drive.Status.AllocatedCapacity)
		}
	}

	deleteVolumeRequests := []csi.DeleteVolumeRequest{
		// DeleteVolumeRequests for volumes in Node N1
		createDeleteVolumeRequest("volume-1"),
		createDeleteVolumeRequest("volume-2"),
		// DeleteVolumeRequests for volumes in Node N2
		createDeleteVolumeRequest("volume-3"),
		createDeleteVolumeRequest("volume-4"),
	}

	for _, dvReq := range deleteVolumeRequests {
		if _, err := cl.DeleteVolume(ctx, &dvReq); err != nil {
			t.Errorf("[%s] DeleteVolume failed: %v", dvReq.VolumeId, err)
		}
	}
}

func TestSelectDriveByFreeCapacity(t1 *testing.T) {
	testCases := []struct {
		name               string
		driveList          []directcsi.DirectCSIDrive
		expectedDriveNames []string
	}{
		{
			name: "test1",
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 1000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 2000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 3000,
					},
				},
			},
			expectedDriveNames: []string{"drive3"},
		},
		{
			name: "test2",
			driveList: []directcsi.DirectCSIDrive{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 4000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 4000,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "drive3",
					},
					Status: directcsi.DirectCSIDriveStatus{
						FreeCapacity: 3000,
					},
				},
			},
			expectedDriveNames: []string{"drive1", "drive2"},
		},
	}

	checkDriveName := func(expectedDriveNames []string, driveName string) bool {
		for _, edName := range expectedDriveNames {
			if edName == driveName {
				return true
			}
		}
		return false
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			selectedDrive, err := selectDriveByFreeCapacity(tt.driveList)
			if err != nil {
				t1.Fatalf("Text case name: %s: Error: %v", tt.name, err)
			}
			if !checkDriveName(tt.expectedDriveNames, selectedDrive.Name) {
				t1.Errorf("Test case name %s: Unexpected drive selected. Expected one among %v but got %s", tt.name, tt.expectedDriveNames, selectedDrive.Name)
			}
		})
	}
}
