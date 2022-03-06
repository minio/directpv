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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	client.FakeInit()
}

const (
	mb100 = 100 * 1024 * 1024
	mb20  = 20 * 1024 * 1024
)

func createFakeController() *ControllerServer {
	return &ControllerServer{
		NodeID:          "test-node-1",
		Identity:        "test-identity-1",
		Rack:            "test-rack-1",
		Zone:            "test-zone-1",
		Region:          "test-region-1",
		directcsiClient: clientsetfake.NewSimpleClientset(),
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
				Filesystem:        "xfs",
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
	cl.directcsiClient = clientsetfake.NewSimpleClientset(testDriveObjects...)
	directCSIClient := cl.directcsiClient.DirectV1beta4()
	client.SetLatestDirectCSIDriveInterface(directCSIClient.DirectCSIDrives())

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
	client.SetLatestDirectCSIDriveInterface(directCSIClient.DirectCSIDrives())
	driveList, err := client.GetDriveList(ctx, nil, nil, nil)
	if err != nil {
		t.Errorf("Listing drives failed: %v", err)
	}

	// Checks to ensure if the volumes were equally distributed among drives
	// And also check if the drive status were updated properly
	for _, drive := range driveList {
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

func TestAbnormalDeleteVolume(t1 *testing.T) {

	testVolumeObjects := []runtime.Object{
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-1",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPVProtection),
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      "node-1",
				Drive:         "test-drive",
				TotalCapacity: int64(100),
				ContainerPath: "",
				StagingPath:   "/path/stagingpath",
				UsedCapacity:  int64(50),
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
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-2",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPVProtection),
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      "node-1",
				Drive:         "test-drive",
				TotalCapacity: int64(100),
				StagingPath:   "/path/stagingpath",
				ContainerPath: "/path/containerpath",
				UsedCapacity:  int64(50),
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
						Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIVolumeConditionReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonReady),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	deleteVolumeRequests := []csi.DeleteVolumeRequest{
		{
			VolumeId: "test-volume-1",
		},
		{
			VolumeId: "test-volume-2",
		},
	}

	ctx := context.TODO()
	cl := createFakeController()
	cl.directcsiClient = clientsetfake.NewSimpleClientset(testVolumeObjects...)
	for _, dvReq := range deleteVolumeRequests {
		if _, err := cl.DeleteVolume(ctx, &dvReq); err == nil {
			t1.Errorf("[%s] DeleteVolume expected to fail but succeeded", dvReq.VolumeId)
		}
	}
}

func TestControllerGetCapabilities(t *testing.T) {
	result, err := createFakeController().ControllerGetCapabilities(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedResult := &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME},
				},
			},
		},
	}
	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v, got: %#+v\n", expectedResult, result)
	}
}

func TestValidateVolumeCapabilities(t *testing.T) {
	testCases := []struct {
		request        *csi.ValidateVolumeCapabilitiesRequest
		expectedResult *csi.ValidateVolumeCapabilitiesResponse
	}{
		{
			&csi.ValidateVolumeCapabilitiesRequest{},
			&csi.ValidateVolumeCapabilitiesResponse{Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{}},
		},
		{
			&csi.ValidateVolumeCapabilitiesRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}},
				},
			},
			&csi.ValidateVolumeCapabilitiesResponse{
				Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
					VolumeCapabilities: []*csi.VolumeCapability{
						{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER}},
					},
				},
			},
		},
		{
			&csi.ValidateVolumeCapabilitiesRequest{
				VolumeCapabilities: []*csi.VolumeCapability{
					{AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER}},
				},
			},
			&csi.ValidateVolumeCapabilitiesResponse{
				Message: "unsupported access mode MULTI_NODE_MULTI_WRITER",
			},
		},
	}

	controller := createFakeController()
	for i, testCase := range testCases {
		result, err := controller.ValidateVolumeCapabilities(context.TODO(), testCase.request)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %#+v, got: %#+v\n", i+1, testCase.expectedResult, result)
		}
	}
}

func TestListVolumes(t *testing.T) {
	if _, err := createFakeController().ListVolumes(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerPublishVolume(t *testing.T) {
	if _, err := createFakeController().ControllerPublishVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	if _, err := createFakeController().ControllerUnpublishVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerExpandVolume(t *testing.T) {
	if _, err := createFakeController().ControllerExpandVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerGetVolume(t *testing.T) {
	if _, err := createFakeController().ControllerGetVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestListSnapshots(t *testing.T) {
	if _, err := createFakeController().ListSnapshots(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestCreateSnapshot(t *testing.T) {
	if _, err := createFakeController().CreateSnapshot(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	if _, err := createFakeController().DeleteSnapshot(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestGetCapacity(t *testing.T) {
	if _, err := createFakeController().GetCapacity(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}
