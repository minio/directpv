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
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/matcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const GiB = 1073741824

func TestGetFilteredDrives(t *testing.T) {
	case2Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "drive-1",
				Finalizers: []string{directcsi.DirectCSIDriveFinalizerPrefix + "volume-1"},
			},
			Status: directcsi.DirectCSIDriveStatus{
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case2Objects := []runtime.Object{&case2Result[0]}
	case2Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	case3Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case3Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case3Result[0],
		&case3Result[1],
	}
	case3Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	case4Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus:  directcsi.DriveStatusReady,
				FreeCapacity: 4 * GiB,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus:  directcsi.DriveStatusInUse,
				FreeCapacity: 2 * GiB,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case4Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case4Result[0],
		&case4Result[1],
	}
	case4Request := &csi.CreateVolumeRequest{Name: "volume-1", CapacityRange: &csi.CapacityRange{RequiredBytes: 2 * GiB}}

	case5Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus:  directcsi.DriveStatusReady,
				FreeCapacity: 4 * GiB,
				Filesystem:   "xfs",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case5Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus:  directcsi.DriveStatusInUse,
				FreeCapacity: 2 * GiB,
				Filesystem:   "ext4",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case5Result[0],
	}
	case5Request := &csi.CreateVolumeRequest{
		Name:               "volume-1",
		CapacityRange:      &csi.CapacityRange{RequiredBytes: 2 * GiB},
		VolumeCapabilities: []*csi.VolumeCapability{{AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "xfs"}}}},
	}

	case6Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				AccessTier:  directcsi.AccessTierHot,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case6Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case6Result[0],
	}
	case6Request := &csi.CreateVolumeRequest{
		Name:       "volume-1",
		Parameters: map[string]string{"direct-csi-min-io/access-tier": string(directcsi.AccessTierHot)},
	}

	case7Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case7Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case7Result[0],
	}
	case7Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Preferred: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"}}},
		},
	}

	case8Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case8Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&case8Result[0],
	}
	case8Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"}}},
		},
	}

	case9Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIDriveReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}
	case9Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Preferred: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"}}},
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"}}},
		},
	}

	case10Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionTrue,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
	}
	case10Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionTrue,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionFalse,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonNotReady),
					},
				},
			},
		},
		&case10Result[0],
	}
	case10Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"}}},
		},
	}

	case11Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionFalse,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionFalse,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonNotReady),
					},
				},
			},
		},
	}
	case11Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"}}},
		},
	}

	case12Result := []directcsi.DirectCSIDrive{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusReady,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionTrue,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
	}

	case12Objects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive-1",
			},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusAvailable,
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionTrue,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{
				Name: "drive-3",
				DeletionTimestamp: func() *metav1.Time {
					deleteTz := metav1.Now()
					return &deleteTz
				}(),
			},
			Status: directcsi.DirectCSIDriveStatus{
				DriveStatus: directcsi.DriveStatusInUse,
				Topology:    map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"},
				Conditions: []metav1.Condition{
					{
						Type:    string(directcsi.DirectCSIDriveConditionReady),
						Status:  metav1.ConditionTrue,
						Message: "",
						Reason:  string(directcsi.DirectCSIDriveReasonReady),
					},
				},
			},
		},
		&case10Result[0],
	}
	case12Request := &csi.CreateVolumeRequest{
		Name: "volume-1",
		AccessibilityRequirements: &csi.TopologyRequirement{
			Requisite: []*csi.Topology{{Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R2"}}},
		},
	}

	testCases := []struct {
		objects        []runtime.Object
		request        *csi.CreateVolumeRequest
		expectedResult []directcsi.DirectCSIDrive
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
	}

	for i, testCase := range testCases {
		client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testCase.objects...).DirectV1beta3().DirectCSIDrives())
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
	case2Result := &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "drive-1",
			Finalizers: []string{directcsi.DirectCSIDriveFinalizerPrefix + "volume-1"},
		},
	}
	case2Objects := []runtime.Object{case2Result}
	case2Request := &csi.CreateVolumeRequest{Name: "volume-1"}

	testCases := []struct {
		objects        []runtime.Object
		request        *csi.CreateVolumeRequest
		expectedResult *directcsi.DirectCSIDrive
		expectErr      bool
	}{
		{[]runtime.Object{}, nil, nil, true},
		{case2Objects, case2Request, case2Result, false},
	}

	for i, testCase := range testCases {
		client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testCase.objects...).DirectV1beta3().DirectCSIDrives())
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
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
			Status:     directcsi.DirectCSIDriveStatus{DriveStatus: directcsi.DriveStatusAvailable},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-2"},
			Status:     directcsi.DirectCSIDriveStatus{DriveStatus: directcsi.DriveStatusReady, FreeCapacity: 4 * GiB},
		},
		&directcsi.DirectCSIDrive{
			ObjectMeta: metav1.ObjectMeta{Name: "drive-3"},
			Status:     directcsi.DirectCSIDriveStatus{DriveStatus: directcsi.DriveStatusInUse, FreeCapacity: 4 * GiB},
		},
	}
	request := &csi.CreateVolumeRequest{Name: "volume-1", CapacityRange: &csi.CapacityRange{RequiredBytes: 2 * GiB}}

	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(objects...).DirectV1beta3().DirectCSIDrives())
	result, err := selectDrive(context.TODO(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matcher.StringIn([]string{"drive-2", "drive-3"}, result.Name) {
		t.Fatalf("result: expected: %v, got: %v", []string{"drive-2", "drive-3"}, result.Name)
	}
}
