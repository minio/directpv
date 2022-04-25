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

package node

import (
	"context"
	"reflect"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestNodeGetVolumeStats(t *testing.T) {
	testObjects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-drive",
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-1",
			},
			Status: directcsi.DirectCSIVolumeStatus{
				Drive:         "test-drive",
				ContainerPath: "/unknown/mnt",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIVolumeConditionStaged),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionPublished),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionAbnormal),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonNormal),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-2",
			},
			Status: directcsi.DirectCSIVolumeStatus{
				Drive:         "test-drive",
				ContainerPath: "/unknown/mnt",
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIVolumeConditionStaged),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionPublished),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionAbnormal),
						Status:             metav1.ConditionTrue,
						Message:            "path /unknown/mnt is not mounted",
						Reason:             string(directcsi.DirectCSIVolumeReasonContainerPathNotMounted),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	testCases := []struct {
		request          *csi.NodeGetVolumeStatsRequest
		expectedResponse *csi.NodeGetVolumeStatsResponse
	}{
		{
			request: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   "test-volume-1",
				VolumePath: "/unknown/mnt",
			},
			expectedResponse: &csi.NodeGetVolumeStatsResponse{
				Usage: []*csi.VolumeUsage{
					{
						Unit: csi.VolumeUsage_BYTES,
					},
				},
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: false,
					Message:  "",
				},
			},
		},
		{
			request: &csi.NodeGetVolumeStatsRequest{
				VolumeId:   "test-volume-2",
				VolumePath: "/unknown/mnt",
			},
			expectedResponse: &csi.NodeGetVolumeStatsResponse{
				Usage: []*csi.VolumeUsage{
					{},
				},
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: true,
					Message:  "path /unknown/mnt is not mounted",
				},
			},
		},
	}
	nodeServer := createFakeNodeServer()
	nodeServer.directcsiClient = fakedirect.NewSimpleClientset(testObjects...)
	ctx := context.TODO()
	for i, testCase := range testCases {
		response, err := nodeServer.NodeGetVolumeStats(ctx, testCase.request)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}
		if !reflect.DeepEqual(response, testCase.expectedResponse) {
			t.Fatalf("case %v: expected: %#+v, got: %#+v\n", i+1, testCase.expectedResponse, response)
		}
		if response.GetVolumeCondition().GetAbnormal() {
			volObj, gErr := nodeServer.directcsiClient.DirectV1beta4().DirectCSIVolumes().Get(ctx, testCase.request.GetVolumeId(), metav1.GetOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			})
			if gErr != nil {
				t.Fatalf("case %v: volume (%s) not found. Error: %v", i+1, testCase.request.GetVolumeId(), gErr)
			}
			if !utils.IsCondition(volObj.Status.Conditions,
				string(directcsi.DirectCSIVolumeConditionAbnormal),
				metav1.ConditionTrue,
				string(directcsi.DirectCSIVolumeReasonContainerPathNotMounted),
				response.GetVolumeCondition().GetMessage()) {
				t.Fatalf("case: %v: abnormal condition is not set to True for abnormal volumes. volume: %v", i+1, testCase.request.GetVolumeId())
			}
		}
	}
}

func TestNodeGetCapabilities(t *testing.T) {
	result, err := createFakeNodeServer().NodeGetCapabilities(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedResult := &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_VOLUME_CONDITION,
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v, got: %#+v\n", expectedResult, result)
	}
}

func TestNodeGetInfo(t *testing.T) {
	ns := createFakeNodeServer()
	result, err := createFakeNodeServer().NodeGetInfo(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedResult := &csi.NodeGetInfoResponse{
		NodeId:            ns.NodeID,
		MaxVolumesPerNode: int64(100),
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				string(utils.TopologyDriverIdentity): ns.Identity,
				string(utils.TopologyDriverRack):     ns.Rack,
				string(utils.TopologyDriverZone):     ns.Zone,
				string(utils.TopologyDriverRegion):   ns.Region,
				string(utils.TopologyDriverNode):     ns.NodeID,
			},
		},
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v, got: %#+v\n", expectedResult, result)
	}
}

func TestNodeExpandVolume(t *testing.T) {
	if _, err := createFakeNodeServer().NodeExpandVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}
