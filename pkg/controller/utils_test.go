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

package controller

import (
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const GiB = 1024 * 1024 * 1024

func TestGetFilteredDrives(t *testing.T) {
	newDriveWithLabels := func(driveID directpvtypes.DriveID, status types.DriveStatus, nodeID directpvtypes.NodeID, driveName directpvtypes.DriveName, labels map[directpvtypes.LabelKey]directpvtypes.LabelValue) *types.Drive {
		drive := types.NewDrive(
			driveID,
			status,
			nodeID,
			driveName,
			directpvtypes.AccessTierDefault,
		)
		for k, v := range labels {
			drive.SetLabel(k, v)
		}
		return drive
	}

	case2Result := []types.Drive{
		*types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
	}
	case2Objects := []runtime.Object{&case2Result[0]}
	case2Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	case3Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierDefault,
		),
		*types.NewDrive(
			"drive-3",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
	}
	case3Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		&case3Result[0],
		&case3Result[1],
	}
	case3Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	case4Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusReady,
				FreeCapacity: 4 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierDefault,
		),
		*types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusReady,
				FreeCapacity: 2 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
	}
	case4Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		&case4Result[0],
		&case4Result[1],
	}
	case4Request := &csi.CreateVolumeRequest{Name: "volume-1", CapacityRange: &csi.CapacityRange{RequiredBytes: 2 * GiB}}

	case5Result := []types.Drive{
		case4Result[0],
	}
	case5Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusLost,
				FreeCapacity: 2 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		&case5Result[0],
	}
	case5Request := &csi.CreateVolumeRequest{
		Name:               "volume-1",
		CapacityRange:      &csi.CapacityRange{RequiredBytes: 2 * GiB},
		VolumeCapabilities: []*csi.VolumeCapability{{AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "xfs"}}}},
	}

	case6Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierHot,
		),
	}
	case6Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusLost,
				FreeCapacity: 2 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		&case6Result[0],
	}
	case6Request := &csi.CreateVolumeRequest{
		Name:       "volume-1",
		Parameters: map[string]string{consts.GroupName + "/access-tier": string(directpvtypes.AccessTierHot)},
	}

	case7Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierHot,
		),
	}
	case7Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		&case7Result[0],
	}
	case7Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Preferred: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"}}},
		},
	}

	case8Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"},
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierHot,
		),
	}
	case8Objects := []runtime.Object{
		types.NewDrive(
			"drive-20",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierHot,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"},
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierHot,
		),
	}
	case8Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"}}},
		},
	}

	case9Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		&case3Result[0],
	}
	case9Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Preferred: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"}}},
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"}}},
		},
	}

	case10Result := []types.Drive{
		*types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"},
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierDefault,
		),
	}
	case10Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusLost,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"},
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		&case10Result[0],
	}
	case10Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"}}},
		},
	}

	case11Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		case10Objects[1],
	}
	case11Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"}}},
		},
	}

	case12Result := []types.Drive{
		case10Result[0],
	}

	deletionTimestamp := metav1.Now()
	drive := types.NewDrive(
		"drive-3",
		types.DriveStatus{
			Status:   directpvtypes.DriveStatusLost,
			Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"},
		},
		"node-1",
		directpvtypes.DriveName("sdc"),
		directpvtypes.AccessTierDefault,
	)
	drive.DeletionTimestamp = &deletionTimestamp
	case12Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		drive,
		&case10Result[0],
	}
	case12Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region2"}}},
		},
	}

	case13Result := []types.Drive{
		case7Result[0],
	}
	case13Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{Status: directpvtypes.DriveStatusLost},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		&case13Result[0],
	}
	case13Request := &csi.CreateVolumeRequest{
		Name:       "volume-1",
		Parameters: map[string]string{consts.GroupName + "/access-tier": "hot"},
	}

	case14Result := []types.Drive{
		*newDriveWithLabels(
			"drive-4",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
			},
			"node-1",
			directpvtypes.DriveName("sdd"),
			map[directpvtypes.LabelKey]directpvtypes.LabelValue{
				consts.GroupName + "/access-type": "hot",
			},
		),
	}

	case14Objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{Status: directpvtypes.DriveStatusReady},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
		newDriveWithLabels(
			"drive-4",
			types.DriveStatus{
				Status:   directpvtypes.DriveStatusReady,
				Topology: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
			},
			"node-1",
			directpvtypes.DriveName("sdd"),
			map[directpvtypes.LabelKey]directpvtypes.LabelValue{
				consts.GroupName + "/access-type": "hot",
			},
		),
	}

	case14Request := &csi.CreateVolumeRequest{
		Name:       "volume-1",
		Parameters: map[string]string{consts.GroupName + "/access-type": "hot"},
	}

	testCases := []struct {
		objects        []runtime.Object
		request        *csi.CreateVolumeRequest
		expectedResult []types.Drive
	}{
		{[]runtime.Object{}, nil, nil},
		{case2Objects, case2Request, case2Result},
		{case3Objects, case3Request, case3Result},
		{case4Objects, case4Request, case4Result},
		{case5Objects, case5Request, case5Result},
		{case6Objects, case6Request, case6Result},
		{case7Objects, case7Request, case7Result},
		{case8Objects, case8Request, case8Result},
		{case9Objects, case9Request, nil},
		{case10Objects, case10Request, case10Result},
		{case11Objects, case11Request, nil},
		{case12Objects, case12Request, case12Result},
		{case13Objects, case13Request, case13Result},
		{case14Objects, case14Request, case14Result},
	}

	for i, testCase := range testCases {
		clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testCase.objects...))
		client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
		result, err := getFilteredDrives(context.TODO(), testCase.request)
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestGetDrive(t *testing.T) {
	case2Result := types.NewDrive(
		"drive-1",
		types.DriveStatus{},
		"node-1",
		directpvtypes.DriveName("sda"),
		directpvtypes.AccessTierDefault,
	)
	case2Result.AddVolumeFinalizer("volume-1")
	case2Objects := []runtime.Object{case2Result}
	case2Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	testCases := []struct {
		objects        []runtime.Object
		request        *csi.CreateVolumeRequest
		expectedResult *types.Drive
		expectErr      bool
	}{
		{[]runtime.Object{}, nil, nil, true},
		{case2Objects, case2Request, case2Result, false},
	}

	for i, testCase := range testCases {
		clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testCase.objects...))
		client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
		client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

		result, err := selectDrive(context.TODO(), testCase.request)

		if testCase.expectErr {
			if err == nil {
				t.Fatalf("case %v: expected error, but succeeded", i+1)
			}
			continue
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: result: expected: %v, got: %v", i+1, testCase.expectedResult, result)
		}
	}

	objects := []runtime.Object{
		types.NewDrive(
			"drive-1",
			types.DriveStatus{Status: directpvtypes.DriveStatusError},
			"node-1",
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-2",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusReady,
				FreeCapacity: 4 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdb"),
			directpvtypes.AccessTierDefault,
		),
		types.NewDrive(
			"drive-3",
			types.DriveStatus{
				Status:       directpvtypes.DriveStatusReady,
				FreeCapacity: 4 * GiB,
			},
			"node-1",
			directpvtypes.DriveName("sdc"),
			directpvtypes.AccessTierDefault,
		),
	}
	request := &csi.CreateVolumeRequest{Name: "volume-1", CapacityRange: &csi.CapacityRange{RequiredBytes: 2 * GiB}}

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	result, err := selectDrive(context.TODO(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !utils.Contains([]string{"drive-2", "drive-3"}, result.Name) {
		t.Fatalf("result: expected: %v, got: %v", []string{"drive-2", "drive-3"}, result.Name)
	}
}
