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
	"github.com/google/uuid"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	client.FakeInit()
}

const MiB = 1024 * 1024

func TestCreateAndDeleteVolumeRPCs(t *testing.T) {
	getTopologySegmentsForNode := func(node string) map[string]string {
		switch node {
		case "node1":
			return map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"}
		case "node2":
			return map[string]string{"node": "node2", "rack": "rack2", "zone": "zone2", "region": "region2"}
		default:
			return map[string]string{}
		}
	}

	createTestDrive100MB := func(nodeID directpvtypes.NodeID, driveID directpvtypes.DriveID) *types.Drive {
		return types.NewDrive(
			driveID,
			types.DriveStatus{
				TotalCapacity: 100 * MiB,
				FreeCapacity:  100 * MiB,
				FSUUID:        string(driveID),
				Status:        directpvtypes.DriveStatusReady,
				Topology:      getTopologySegmentsForNode(string(nodeID)),
			},
			nodeID,
			directpvtypes.DriveName("sda"),
			directpvtypes.AccessTierDefault,
		)
	}

	create20MBVolumeRequest := func(volName string, requestedNode string) csi.CreateVolumeRequest {
		return csi.CreateVolumeRequest{
			Name: volName,
			CapacityRange: &csi.CapacityRange{
				RequiredBytes: 20 * MiB,
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
						Segments: map[string]string{"node": "node1", "rack": "rack1", "zone": "zone1", "region": "region1"},
					},
					{
						Segments: map[string]string{"node": "node2", "rack": "rack2", "zone": "zone2", "region": "region2"},
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
		// Drives from node1
		createTestDrive100MB("node1", "D1"),
		createTestDrive100MB("node1", "D2"),
		// Drives from node2
		createTestDrive100MB("node2", "D3"),
		createTestDrive100MB("node2", "D4"),
	}

	createVolumeRequests := []csi.CreateVolumeRequest{
		// Volume requests for drives in node1
		create20MBVolumeRequest("volume-1", "node1"),
		create20MBVolumeRequest("volume-2", "node1"),
		// Volume requests for drives in node2
		create20MBVolumeRequest("volume-3", "node2"),
		create20MBVolumeRequest("volume-4", "node2"),
	}

	ctx := context.TODO()
	cl := NewServer()

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testDriveObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

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
		volObj, gErr := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		})
		if gErr != nil {
			t.Fatalf("[%s] Volume (%s) not found. Error: %v", volName, volumeID, gErr)
		}
		// Step 5: Check if the finalizers were set correctly
		volFinalizers := volObj.GetFinalizers()
		for _, f := range volFinalizers {
			switch f {
			case consts.GroupName + "/pv-protection":
			case consts.GroupName + "/purge-protection":
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
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	driveList, err := drive.NewLister().Get(ctx)
	if err != nil {
		t.Errorf("Listing drives failed: %v", err)
	}

	// Checks to ensure if the volumes were equally distributed among drives
	// And also check if the drive status were updated properly
	for _, drive := range driveList {
		if len(drive.GetFinalizers()) > 2 {
			t.Errorf("Volumes were not equally distributed among drives. Drive name: %v Finaliers: %v", drive.Name, drive.GetFinalizers())
		}
		if drive.Status.Status != directpvtypes.DriveStatusReady {
			t.Errorf("Expected drive(%s) status: %s, But got: %v", drive.Name, string(directpvtypes.DriveStatusReady), string(drive.Status.Status))
		}
		if drive.Status.FreeCapacity != (100*MiB - 20*MiB) {
			t.Errorf("Expected drive(%s) FreeCapacity: %d, But got: %d", drive.Name, (100*MiB - 20*MiB), drive.Status.FreeCapacity)
		}
		if drive.Status.AllocatedCapacity != 20*MiB {
			t.Errorf("Expected drive(%s) AllocatedCapacity: %d, But got: %d", drive.Name, 20*MiB, drive.Status.AllocatedCapacity)
		}
	}

	deleteVolumeRequests := []csi.DeleteVolumeRequest{
		// DeleteVolumeRequests for volumes in node1
		createDeleteVolumeRequest("volume-1"),
		createDeleteVolumeRequest("volume-2"),
		// DeleteVolumeRequests for volumes in node2
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
	fsuuid := uuid.NewString()
	volume1 := types.NewVolume(
		"test-volume-1",
		fsuuid,
		"node-1",
		directpvtypes.DriveID(fsuuid),
		"test-drive",
		100,
	)
	volume1.Status.StagingTargetPath = "/path/staging/targetpath"
	volume1.Status.UsedCapacity = 50

	volume2 := *volume1
	volume2.Name = "test-volume-2"
	volume2.Status.TargetPath = "/path/targetpath"

	testVolumeObjects := []runtime.Object{volume1, &volume2}

	deleteVolumeRequests := []csi.DeleteVolumeRequest{
		{
			VolumeId: "test-volume-1",
		},
		{
			VolumeId: "test-volume-2",
		},
	}

	ctx := context.TODO()
	cl := NewServer()

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVolumeObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	for _, dvReq := range deleteVolumeRequests {
		if _, err := cl.DeleteVolume(ctx, &dvReq); err == nil {
			t1.Errorf("[%s] DeleteVolume expected to fail but succeeded", dvReq.VolumeId)
		}
	}
}

func TestControllerGetCapabilities(t *testing.T) {
	result, err := NewServer().ControllerGetCapabilities(context.TODO(), nil)
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
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_EXPAND_VOLUME},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_GET_VOLUME},
				},
			},
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_VOLUME_CONDITION},
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

	controller := NewServer()
	for i, testCase := range testCases {
		result, err := controller.ValidateVolumeCapabilities(context.TODO(), testCase.request)
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %#+v, got: %#+v\n", i+1, testCase.expectedResult, result)
		}
	}
}

func TestListVolumes(t *testing.T) {
	testObjects := []runtime.Object{
		&types.Drive{
			TypeMeta: types.NewDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-drive",
			},
			Status: types.DriveStatus{
				Topology: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-abnormal-volume-1",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionTrue,
						Message:            string(directpvtypes.VolumeConditionMessageStagingPathNotMounted),
						Reason:             string(directpvtypes.VolumeConditionReasonNotMounted),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-abnormal-volume-2",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionTrue,
						Message:            string(directpvtypes.VolumeConditionMessageTargetPathNotMounted),
						Reason:             string(directpvtypes.VolumeConditionReasonNotMounted),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-normal-volume-1",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	ctx := context.TODO()
	cl := NewServer()
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	getListVolumeResponseEntry := func(volumeId string, abnormal bool, message string) *csi.ListVolumesResponse_Entry {
		return &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				CapacityBytes: int64(100),
				VolumeId:      volumeId,
				AccessibleTopology: []*csi.Topology{
					{
						Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
					},
				},
			},
			Status: &csi.ListVolumesResponse_VolumeStatus{
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: abnormal,
					Message:  message,
				},
			},
		}
	}

	expectedListVolumeResponseEntries := []*csi.ListVolumesResponse_Entry{
		getListVolumeResponseEntry("test-abnormal-volume-1", true, string(directpvtypes.VolumeConditionMessageStagingPathNotMounted)),
		getListVolumeResponseEntry("test-abnormal-volume-2", true, string(directpvtypes.VolumeConditionMessageTargetPathNotMounted)),
		getListVolumeResponseEntry("test-normal-volume-1", false, ""),
	}

	req := &csi.ListVolumesRequest{
		MaxEntries:    int32(3),
		StartingToken: "",
	}
	listVolumesRes, err := cl.ListVolumes(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	listVolumeResponseEntries := listVolumesRes.GetEntries()
	if !reflect.DeepEqual(listVolumeResponseEntries, expectedListVolumeResponseEntries) {
		t.Fatalf("expected volume response entries: %#+v, got: %#+v\n", expectedListVolumeResponseEntries, listVolumeResponseEntries)
	}
}

func TestControllerPublishVolume(t *testing.T) {
	if _, err := NewServer().ControllerPublishVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	if _, err := NewServer().ControllerUnpublishVolume(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestControllerExpandVolume(t *testing.T) {
	volumeID := "test-volume-1"
	reqs := []csi.ControllerExpandVolumeRequest{
		{
			VolumeId:      volumeID,
			CapacityRange: &csi.CapacityRange{RequiredBytes: 50},
		},
		{
			VolumeId:      volumeID,
			CapacityRange: &csi.CapacityRange{RequiredBytes: 150},
		},
	}

	driveID := directpvtypes.DriveID(uuid.NewString())
	nodeID := directpvtypes.NodeID("node-1")
	driveName := directpvtypes.DriveName("sda")
	drive := types.NewDrive(
		driveID,
		types.DriveStatus{
			TotalCapacity: 100 * MiB,
			FreeCapacity:  100 * MiB,
			FSUUID:        string(driveID),
			Status:        directpvtypes.DriveStatusReady,
			Topology:      map[string]string{},
		},
		nodeID,
		driveName,
		directpvtypes.AccessTierDefault,
	)

	volume := types.NewVolume(volumeID, uuid.NewString(), nodeID, driveID, driveName, 100)
	volume.Status.StagingTargetPath = "/path/staging/targetpath"
	volume.Status.UsedCapacity = 50

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume, drive))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	ctx := context.TODO()
	server := NewServer()
	for i, req := range reqs {
		if _, err := server.ControllerExpandVolume(ctx, &req); err != nil {
			t.Errorf("case %v: expected: success; but failed by %v", i+1, err)
		}
	}
}

func TestControllerGetVolume(t *testing.T) {
	testObjects := []runtime.Object{
		&types.Drive{
			TypeMeta: types.NewDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-drive",
			},
			Status: types.DriveStatus{
				Topology: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-abnormal-volume-1",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionTrue,
						Message:            string(directpvtypes.VolumeConditionMessageStagingPathNotMounted),
						Reason:             string(directpvtypes.VolumeConditionReasonNotMounted),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-abnormal-volume-2",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionTrue,
						Message:            string(directpvtypes.VolumeConditionMessageTargetPathNotMounted),
						Reason:             string(directpvtypes.VolumeConditionReasonNotMounted),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-normal-volume-1",
				Labels: map[string]string{
					string(directpvtypes.DriveLabelKey):     "test-drive",
					string(directpvtypes.NodeLabelKey):      "N1",
					string(directpvtypes.DriveNameLabelKey): "/dev/test-drive",
					string(directpvtypes.CreatedByLabelKey): consts.ControllerName,
				},
			},
			Status: types.VolumeStatus{
				TotalCapacity: int64(100),
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeError),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	ctx := context.TODO()
	cl := NewServer()
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	getControllerGetVolumeResponse := func(volumeId string, abnormal bool, message string) *csi.ControllerGetVolumeResponse {
		return &csi.ControllerGetVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes: int64(100),
				VolumeId:      volumeId,
				AccessibleTopology: []*csi.Topology{
					{
						Segments: map[string]string{"node": "N1", "rack": "RK1", "zone": "Z1", "region": "R1"},
					},
				},
			},
			Status: &csi.ControllerGetVolumeResponse_VolumeStatus{
				VolumeCondition: &csi.VolumeCondition{
					Abnormal: abnormal,
					Message:  message,
				},
			},
		}
	}

	testCases := []struct {
		req         *csi.ControllerGetVolumeRequest
		expectedRes *csi.ControllerGetVolumeResponse
	}{
		{
			req: &csi.ControllerGetVolumeRequest{
				VolumeId: "test-abnormal-volume-1",
			},
			expectedRes: getControllerGetVolumeResponse("test-abnormal-volume-1", true, string(directpvtypes.VolumeConditionMessageStagingPathNotMounted)),
		},
		{
			req: &csi.ControllerGetVolumeRequest{
				VolumeId: "test-abnormal-volume-2",
			},
			expectedRes: getControllerGetVolumeResponse("test-abnormal-volume-2", true, string(directpvtypes.VolumeConditionMessageTargetPathNotMounted)),
		},
		{
			req: &csi.ControllerGetVolumeRequest{
				VolumeId: "test-normal-volume-1",
			},
			expectedRes: getControllerGetVolumeResponse("test-normal-volume-1", false, ""),
		},
	}

	for i, testCase := range testCases {
		result, err := cl.ControllerGetVolume(ctx, testCase.req)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}
		if !reflect.DeepEqual(result, testCase.expectedRes) {
			t.Fatalf("case %v: expected: %#+v, got: %#+v\n", i+1, testCase.expectedRes, result)
		}
	}
}

func TestListSnapshots(t *testing.T) {
	if _, err := NewServer().ListSnapshots(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestCreateSnapshot(t *testing.T) {
	if _, err := NewServer().CreateSnapshot(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestDeleteSnapshot(t *testing.T) {
	if _, err := NewServer().DeleteSnapshot(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}

func TestGetCapacity(t *testing.T) {
	if _, err := NewServer().GetCapacity(context.TODO(), nil); err == nil {
		t.Fatal("error expected")
	}
}
