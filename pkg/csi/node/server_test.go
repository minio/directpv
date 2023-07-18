// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestNodeGetInfo(t *testing.T) {
	result, err := createFakeServer().NodeGetInfo(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedResult := &csi.NodeGetInfoResponse{
		NodeId: testNodeName,
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				string(directpvtypes.TopologyDriverIdentity): testIdentityName,
				string(directpvtypes.TopologyDriverRack):     testRackName,
				string(directpvtypes.TopologyDriverZone):     testZoneName,
				string(directpvtypes.TopologyDriverRegion):   testRegionName,
				string(directpvtypes.TopologyDriverNode):     testNodeName,
			},
		},
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v, got: %#+v\n", expectedResult, result)
	}
}

func TestNodeGetCapabilities(t *testing.T) {
	result, err := createFakeServer().NodeGetCapabilities(context.TODO(), nil)
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
						Type: csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
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

func TestNodeGetVolumeStats(t *testing.T) {
	testObjects := []runtime.Object{
		&types.Volume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "volume-1",
				Labels: map[string]string{
					string(directpvtypes.NodeLabelKey):      "test-node",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				StagingTargetPath: "/stagingpath/volume-1",
				TargetPath:        "/targetpath/cvolume-1",
				Conditions:        []metav1.Condition{},
			},
		},
		&types.Volume{
			ObjectMeta: metav1.ObjectMeta{
				Name: "volume-2",
				Labels: map[string]string{
					string(directpvtypes.NodeLabelKey):      "test-node",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				StagingTargetPath: "/stagingpath/volume-2",
				TargetPath:        "/containerpath/volume-2",
				Conditions: []metav1.Condition{
					{
						Type:    string(directpvtypes.VolumeConditionTypeError),
						Status:  metav1.ConditionTrue,
						Reason:  string(directpvtypes.VolumeConditionReasonNotMounted),
						Message: string(directpvtypes.VolumeConditionMessageStagingPathNotMounted),
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
				VolumeId:   "volume-1",
				VolumePath: "/stagingpath/volume-1",
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
				VolumeId:   "volume-2",
				VolumePath: "/stagingpath/volume-2",
			},
			expectedResponse: &csi.NodeGetVolumeStatsResponse{
				Usage: []*csi.VolumeUsage{
					{},
				},
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: true,
					Message:  string(directpvtypes.VolumeConditionMessageStagingPathNotMounted),
				},
			},
		},
	}
	nodeServer := createFakeServer()
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	ctx := context.TODO()
	for i, testCase := range testCases {
		response, err := nodeServer.NodeGetVolumeStats(ctx, testCase.request)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}
		if !reflect.DeepEqual(response, testCase.expectedResponse) {
			t.Fatalf("case %v: expected: %#+v, got: %#+v\n", i+1, testCase.expectedResponse, response)
		}
	}
}

func TestNodeExpandVolume(t *testing.T) {
	volumeID := "volume-id-1"
	volume := types.NewVolume(volumeID, "fsuuid1", "node-1", "drive-1", "sda", 100*MiB)
	volume.Status.DataPath = "volume/id/1/data/path"
	volume.Status.StagingTargetPath = "volume/id/1/staging/target/path"

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	nodeServer := createFakeServer()
	nodeServer.getDeviceByFSUUID = func(fsuuid string) (string, error) {
		return "sda", nil
	}
	nodeServer.setQuota = func(ctx context.Context, device, dataPath, name string, quota xfs.Quota, update bool) error {
		return nil
	}

	if _, err := nodeServer.NodeExpandVolume(context.TODO(), &csi.NodeExpandVolumeRequest{
		VolumeId:      volumeID,
		VolumePath:    "volume-id-1-volume-path",
		CapacityRange: &csi.CapacityRange{RequiredBytes: 100 * MiB},
	}); err != nil {
		t.Fatal(err)
	}
}
