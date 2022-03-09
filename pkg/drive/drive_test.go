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

package drive

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	testNodeID = "test-node"
)

func createFakeDriveEventListener() *driveEventHandler {
	return &driveEventHandler{
		nodeID:          testNodeID,
		getDevice:       func(major, minor uint32) (string, error) { return "", nil },
		stat:            func(name string) (os.FileInfo, error) { return nil, nil },
		mountDevice:     func(device, target string, flags []string) error { return nil },
		unmountDevice:   func(device string) error { return nil },
		makeFS:          func(ctx context.Context, device, uuid string, force, reflink bool) error { return nil },
		getFreeCapacity: func(path string) (uint64, error) { return 0, nil },
	}
}

func TestUpdateDriveNoOp(t *testing.T) {
	dl := createFakeDriveEventListener()
	b := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test_drive",
		},
		Spec:   directcsi.DirectCSIDriveSpec{},
		Status: directcsi.DirectCSIDriveStatus{},
	}
	ctx := context.TODO()
	err := dl.update(ctx, &b)
	if err != nil {
		t.Errorf("Error returned [NoOP]: %+v", err)
	}
}

// Sets the requested format in the Spec and checks if desired results are seen.
func TestDriveFormat(t *testing.T) {

	testDriveObjs := []runtime.Object{
		&directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_umounted_uuid",
			},
			Spec: directcsi.DirectCSIDriveSpec{
				DirectCSIOwned: false,
			},
			Status: directcsi.DirectCSIDriveStatus{
				NodeName:       testNodeID,
				DriveStatus:    directcsi.DriveStatusAvailable,
				Path:           "/drive/path",
				FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				MajorNumber:    202,
				MinorNumber:    1,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionOwned),
						Status:             metav1.ConditionFalse,
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionMounted),
						Status:             metav1.ConditionFalse,
						Message:            "not mounted",
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionFormatted),
						Status:             metav1.ConditionFalse,
						Message:            "xfs",
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionInitialized),
						Status:             metav1.ConditionTrue,
						Message:            "initialized",
						Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
						LastTransitionTime: metav1.Now(),
					},
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
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_mounted_uuid",
			},
			Spec: directcsi.DirectCSIDriveSpec{
				DirectCSIOwned: false,
			},
			Status: directcsi.DirectCSIDriveStatus{
				NodeName:       testNodeID,
				DriveStatus:    directcsi.DriveStatusAvailable,
				Path:           "/drive/path",
				Mountpoint:     "/mnt/mp",
				FilesystemUUID: "d8e7d5de-88c6-4675-9e38-f712669e87b3",
				MajorNumber:    202,
				MinorNumber:    2,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIDriveConditionOwned),
						Status:             metav1.ConditionFalse,
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionMounted),
						Status:             metav1.ConditionTrue,
						Message:            "/mnt/mp",
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionFormatted),
						Status:             metav1.ConditionFalse,
						Message:            "xfs",
						Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directcsi.DirectCSIDriveConditionInitialized),
						Status:             metav1.ConditionTrue,
						Message:            "initialized",
						Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
						LastTransitionTime: metav1.Now(),
					},
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

	ctx := context.TODO()

	// Step 1: Construct fake drive listener
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDriveObjs...).DirectV1beta3().DirectCSIDrives())

	for i, tObj := range testDriveObjs {
		dObj := tObj.(*directcsi.DirectCSIDrive)
		// Step 2: Get the object
		newObj, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, dObj.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
		if dErr != nil {
			t.Fatalf("Error while getting the drive object: %+v", dErr)
		}

		// Step 3: Set RequestedFormat to enable formatting
		newObj.Spec.DirectCSIOwned = true
		force := true
		newObj.Spec.RequestedFormat = &directcsi.RequestedFormat{
			Force:      force,
			Filesystem: "xfs",
		}

		// Step 4: Execute the Update hook
		if err := dl.update(ctx, newObj); err != nil {
			t.Errorf("Test case [%d]: Error while invoking the update listener: %+v", i, err)
		}

		// Step 5: Get the latest version of the object
		csiDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, newObj.Name, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if dErr != nil {
			t.Errorf("Test case [%d]: Error while fetching the drive object: %+v", i, dErr)
		}

		// Step 6: Check if the Status fields are updated
		if csiDrive.Status.DriveStatus != directcsi.DriveStatusReady {
			t.Errorf("Test case [%d]: Drive is not in 'ready' state after formatting. Current status: %s", i, csiDrive.Status.DriveStatus)
		}
		if csiDrive.Status.Mountpoint != filepath.Join(sys.MountRoot, newObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Drive mountpoint invalid: %s", i, csiDrive.Status.Mountpoint)
		}
		if csiDrive.Status.Filesystem != "xfs" {
			t.Errorf("Test case [%d]: Invalid filesystem after formatting: %s", i, string(csiDrive.Status.Filesystem))
		}

		// Step 7: Check if the expected conditions are set
		if !utils.IsCondition(csiDrive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionOwned),
			metav1.ConditionTrue,
			string(directcsi.DirectCSIDriveReasonAdded),
			"") {
			t.Errorf("Test case [%d]: unexpected status.condition for %s = %v", i, string(directcsi.DirectCSIDriveConditionOwned), csiDrive.Status.Conditions)
		}
		if !utils.IsCondition(
			csiDrive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionMounted),
			metav1.ConditionTrue,
			string(directcsi.DirectCSIDriveReasonAdded),
			string(directcsi.DirectCSIDriveMessageMounted)) {
			t.Errorf("Test case [%d]: unexpected status.condition for %s = %v", i, string(directcsi.DirectCSIDriveConditionMounted), csiDrive.Status.Conditions)
		}
		if !utils.IsCondition(csiDrive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionFormatted),
			metav1.ConditionTrue,
			string(directcsi.DirectCSIDriveReasonAdded),
			string(directcsi.DirectCSIDriveMessageFormatted)) {
			t.Errorf("Test case [%d]: unexpected status.condition for %s = %v", i, string(directcsi.DirectCSIDriveConditionFormatted), csiDrive.Status.Conditions)
		}
	}
}

func TestDriveDelete(t *testing.T) {
	testCases := []struct {
		name               string
		expectErr          bool
		driveObject        directcsi.DirectCSIDrive
		expectedFinalizers []string
	}{
		{
			name: "testDeletionLifecycleSuccessCase",
			driveObject: directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_drive_1",
					Finalizers: []string{
						string(directcsi.DirectCSIDriveFinalizerDataProtection),
					},
				},
				Spec: directcsi.DirectCSIDriveSpec{
					DirectCSIOwned: false,
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:    testNodeID,
					DriveStatus: directcsi.DriveStatusReady,
					Path:        "/drive/path",
				},
			},
			expectErr:          false,
			expectedFinalizers: []string{},
		},
		{
			name: "testDeletionLifecycleFailureCase",
			driveObject: directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_drive_2",
					Finalizers: []string{
						directcsi.DirectCSIDriveFinalizerPrefix + "vol_id",
						string(directcsi.DirectCSIDriveFinalizerDataProtection),
					},
				},
				Spec: directcsi.DirectCSIDriveSpec{
					DirectCSIOwned: false,
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:    testNodeID,
					DriveStatus: directcsi.DriveStatusInUse,
					Path:        "/drive/path",
				},
			},
			expectErr: true,
			expectedFinalizers: []string{
				directcsi.DirectCSIDriveFinalizerPrefix + "vol_id",
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
		},
	}
	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	fakeDirectCSIClient := clientsetfake.NewSimpleClientset(&testCases[0].driveObject, &testCases[1].driveObject).DirectV1beta3()
	client.SetLatestDirectCSIDriveInterface(fakeDirectCSIClient.DirectCSIDrives())
	client.SetLatestDirectCSIVolumeInterface(fakeDirectCSIClient.DirectCSIVolumes())

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			newObj, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, tt.driveObject.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
			if dErr != nil {
				t.Fatalf("Error while getting the drive object: %+v", dErr)
			}

			now := metav1.Now()
			newObj.ObjectMeta.DeletionTimestamp = &now
			if err := dl.delete(ctx, newObj); err != nil && !tt.expectErr {
				t.Errorf("Error while invoking the update listener: %+v", err)
			}

			csiDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, newObj.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
			if dErr != nil {
				t.Fatalf("Error while fetching the drive object: %+v", dErr)
			}

			if csiDrive.Status.DriveStatus != directcsi.DriveStatusTerminating {
				t.Errorf("Drive is not in 'terminating' state after deletion. Current status: %s", csiDrive.Status.DriveStatus)
			}

			if !reflect.DeepEqual(csiDrive.ObjectMeta.GetFinalizers(), tt.expectedFinalizers) {
				t.Errorf("Expected Finalizers: %v but got: %v", tt.expectedFinalizers, csiDrive.ObjectMeta.GetFinalizers())
			}
		})
	}
}

func TestDriveRelease(t *testing.T) {
	driveObject := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-drive",
			Finalizers: []string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
		},
		Spec: directcsi.DirectCSIDriveSpec{
			DirectCSIOwned: false,
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:    "test-node",
			DriveStatus: directcsi.DriveStatusReady,
			Path:        "/drive/path",
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionTrue,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             metav1.ConditionTrue,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             metav1.ConditionTrue,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string(directcsi.DirectCSIDriveReasonReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(&driveObject).DirectV1beta3().DirectCSIDrives())

	newObj, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, "test-drive", metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
	if dErr != nil {
		t.Fatalf("Error while getting the drive object: %+v", dErr)
	}

	if err := dl.release(ctx, newObj); err != nil {
		t.Errorf("Error while invoking the update listener: %+v", err)
	}

	csiDrive, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, "test-drive", metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
	if dErr != nil {
		t.Fatalf("Error while fetching the drive object: %+v", dErr)
	}

	if csiDrive.Status.DriveStatus != directcsi.DriveStatusAvailable {
		t.Errorf("Drive is not in 'available' state after releasing. Current status: %s", csiDrive.Status.DriveStatus)
	}

	if !utils.IsCondition(csiDrive.Status.Conditions, string(directcsi.DirectCSIDriveConditionOwned), metav1.ConditionFalse, string(directcsi.DirectCSIDriveReasonAdded), "") {
		t.Errorf("Incorrect %s condition in: %v", string(directcsi.DirectCSIDriveConditionOwned), csiDrive.Status.Conditions)
	}

	if !utils.IsCondition(csiDrive.Status.Conditions, string(directcsi.DirectCSIDriveConditionMounted), metav1.ConditionFalse, string(directcsi.DirectCSIDriveReasonAdded), "") {
		t.Errorf("Incorrect %s condition in: %v", string(directcsi.DirectCSIDriveConditionMounted), csiDrive.Status.Conditions)
	}
}

func TestInUseDriveDeletion(t *testing.T) {
	driveObject := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-drive",
			Finalizers: []string{
				directcsi.DirectCSIDriveFinalizerPrefix + "test-volume",
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
		},
		Spec: directcsi.DirectCSIDriveSpec{
			DirectCSIOwned: false,
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:    "node-1",
			DriveStatus: directcsi.DriveStatusInUse,
			Path:        "/drive/path",
		},
	}
	volumeObject := directcsi.DirectCSIVolume{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-volume",
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
	}

	ctx := context.TODO()
	dl := createFakeDriveEventListener()
	fakeDirectCSIClient := clientsetfake.NewSimpleClientset(&driveObject, &volumeObject).DirectV1beta3()
	client.SetLatestDirectCSIDriveInterface(fakeDirectCSIClient.DirectCSIDrives())
	client.SetLatestDirectCSIVolumeInterface(fakeDirectCSIClient.DirectCSIVolumes())

	driveObj, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, "test-drive", metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
	if dErr != nil {
		t.Fatalf("Error while getting the drive object: %+v", dErr)
	}

	now := metav1.Now()
	driveObj.ObjectMeta.DeletionTimestamp = &now
	if err := dl.delete(ctx, driveObj); err != nil {
		t.Errorf("Error while invoking the delete listener: %+v", err)
	}

	volumeObj, volErr := client.GetLatestDirectCSIVolumeInterface().Get(ctx, "test-volume", metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()})
	if volErr != nil {
		t.Fatalf("Error while getting the volume object: %+v", volErr)
	}

	if !utils.IsConditionStatus(volumeObj.Status.Conditions, string(directcsi.DirectCSIVolumeConditionReady), metav1.ConditionFalse) {
		t.Errorf("Incorrect status condition: %v", volumeObj.Status.Conditions)
	}
}
