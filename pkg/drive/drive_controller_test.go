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

package drive

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	testNodeID = "test-node"
)

type fakeDriveStatter struct {
	args struct {
		path string
	}
}

func (c *fakeDriveStatter) GetFreeCapacityFromStatfs(path string) (int64, error) {
	c.args.path = path
	return 0, nil
}

type fakeDriveFormatter struct {
	formatArgs struct {
		uuid  string
		path  string
		force bool
	}
	makeBlockFileArgs struct {
		path  string
		major uint32
		minor uint32
	}
}

func (c *fakeDriveFormatter) FormatDrive(ctx context.Context, uuid, path string, force bool) error {
	c.formatArgs.path = path
	c.formatArgs.force = force
	c.formatArgs.uuid = uuid
	return nil
}

func (c *fakeDriveFormatter) MakeBlockFile(path string, major, minor uint32) error {
	c.makeBlockFileArgs.path = path
	c.makeBlockFileArgs.major = major
	c.makeBlockFileArgs.minor = minor
	return nil
}

type fakeDriveMounter struct {
	mountArgs struct {
		source    string
		target    string
		mountOpts []string
	}
	unmountArgs struct {
		source string
	}
}

func (c *fakeDriveMounter) MountDrive(source, target string, mountOpts []string) error {
	c.mountArgs.source = source
	c.mountArgs.target = target
	c.mountArgs.mountOpts = mountOpts
	return nil
}

func (c *fakeDriveMounter) UnmountDrive(path string) error {
	c.unmountArgs.source = path
	return nil
}

func createFakeDriveListener() *DirectCSIDriveListener {
	utils.SetFake()

	fakeKubeClnt := utils.GetKubeClient()
	fakeDirectCSIClnt := utils.GetDirectClientset()

	return &DirectCSIDriveListener{
		kubeClient:      fakeKubeClnt,
		directcsiClient: fakeDirectCSIClnt,
		nodeID:          testNodeID,
		mounter:         &fakeDriveMounter{},
		formatter:       &fakeDriveFormatter{},
		statter:         &fakeDriveStatter{},
	}
}

func TestAddAndDeleteDriveNoOp(t *testing.T) {
	dl := createFakeDriveListener()
	b := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test_drive",
		},
		Spec:   directcsi.DirectCSIDriveSpec{},
		Status: directcsi.DirectCSIDriveStatus{},
	}
	ctx := context.TODO()
	err := dl.Add(ctx, &b)
	if err != nil {
		t.Errorf("Error returned [Add]: %+v", err)
	}

	err = dl.Delete(ctx, &b)
	if err != nil {
		t.Errorf("Error returned [Delete]: %+v", err)
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
				FilesystemUUID: "test_drive_umounted_uuid",
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
				FilesystemUUID: "test_drive_mounted_uuid",
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
				},
			},
		},
	}

	ctx := context.TODO()

	// Step 1: Construct fake drive listener
	dl := createFakeDriveListener()
	dl.directcsiClient = fakedirect.NewSimpleClientset(testDriveObjs...)
	directCSIClient := dl.directcsiClient.DirectV1beta2()

	for i, tObj := range testDriveObjs {
		dObj := tObj.(*directcsi.DirectCSIDrive)
		// Step 2: Get the object
		newObj, dErr := directCSIClient.DirectCSIDrives().Get(ctx, dObj.Name, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if dErr != nil {
			t.Fatalf("Error while getting the drive object: %+v", dErr)
		}

		// Step 3: Set RequestedFormat to enable formatting
		newObj.Spec.DirectCSIOwned = true
		force := true
		newObj.Spec.RequestedFormat = &directcsi.RequestedFormat{
			Force:      force,
			Filesystem: string(sys.FSTypeXFS),
		}

		// Step 4: Execute the Update hook
		if err := dl.Update(ctx, dObj, newObj); err != nil {
			t.Errorf("Test case [%d]: Error while invoking the update listener: %+v", i, err)
		}

		// Step 4.1: Check if MakeBlockFile arguments passed are correct
		if dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.path != sys.GetDirectCSIPath(dObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Invalid path provided for makeBlockFile call. Expected: %s, Found: %s", i, sys.GetDirectCSIPath(dObj.Status.FilesystemUUID), dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.path)
		}
		if dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.major != dObj.Status.MajorNumber {
			t.Errorf("Test case [%d]: Invalid major number provided for makeBlockFile call. Expected: %v, Found: %v", i, dObj.Status.MajorNumber, dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.major)
		}
		if dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.minor != dObj.Status.MinorNumber {
			t.Errorf("Test case [%d]: Invalid minor number provided for makeBlockFile call. Expected: %v, Found: %v", i, dObj.Status.MinorNumber, dl.formatter.(*fakeDriveFormatter).makeBlockFileArgs.minor)
		}

		// Step 4.1: Check if the format arguments passed are correct
		if dl.formatter.(*fakeDriveFormatter).formatArgs.uuid != dObj.Status.FilesystemUUID {
			t.Errorf("Test case [%d]: Invalid uuid provided for formatting. Expected: %s, Found: %s", i, dObj.Status.FilesystemUUID, dl.formatter.(*fakeDriveFormatter).formatArgs.uuid)
		}
		if dl.formatter.(*fakeDriveFormatter).formatArgs.path != sys.GetDirectCSIPath(dObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Invalid path provided for formatting. Expected: %s, Found: %s", i, sys.GetDirectCSIPath(dObj.Status.FilesystemUUID), dl.formatter.(*fakeDriveFormatter).formatArgs.path)
		}
		if dl.formatter.(*fakeDriveFormatter).formatArgs.force != force {
			t.Errorf("Test case [%d]: Wrong force option provided for formatting. Expected: %v, Found: %v", i, force, dl.formatter.(*fakeDriveFormatter).formatArgs.force)
		}

		// Step 4.2: Check if mount arguments passed are correct
		if dl.mounter.(*fakeDriveMounter).mountArgs.source != sys.GetDirectCSIPath(dObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Invalid source provided for mounting. Expected: %s, Found: %s", i, sys.GetDirectCSIPath(dObj.Status.FilesystemUUID), dl.mounter.(*fakeDriveMounter).mountArgs.source)
		}
		if dl.mounter.(*fakeDriveMounter).mountArgs.target != filepath.Join(sys.MountRoot, dObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Wrong target provided for mounting. Expected: %s, Found: %s", i, filepath.Join(sys.MountRoot, dObj.Status.FilesystemUUID), dl.mounter.(*fakeDriveMounter).mountArgs.target)
		}
		umountSource := func() string {
			if dObj.Status.Mountpoint != "" {
				return sys.GetDirectCSIPath(dObj.Status.FilesystemUUID)
			}
			return ""
		}()
		if dl.mounter.(*fakeDriveMounter).unmountArgs.source != umountSource {
			t.Errorf("Test case [%d]: Invalid source provided for unmounting. Expected: %s, Found: %s", i, umountSource, dl.mounter.(*fakeDriveMounter).unmountArgs.source)
		}

		// Step 4.3: Check if stat arguments passed are correct
		if dl.statter.(*fakeDriveStatter).args.path != filepath.Join(sys.MountRoot, dObj.Status.FilesystemUUID) {
			t.Errorf("Test case [%d]: Wrong path provided for statting. Expected: %s, Found: %s", i, filepath.Join(sys.MountRoot, dObj.Name), dl.statter.(*fakeDriveStatter).args.path)
		}

		// Step 5: Get the latest version of the object
		csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, newObj.Name, metav1.GetOptions{
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
		if csiDrive.Status.Filesystem != string(sys.FSTypeXFS) {
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

func TestUpdateDriveDelete(t *testing.T) {
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
	dl := createFakeDriveListener()
	dl.directcsiClient = fakedirect.NewSimpleClientset(&testCases[0].driveObject, &testCases[1].driveObject)
	directCSIClient := dl.directcsiClient.DirectV1beta2()

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			newObj, dErr := directCSIClient.DirectCSIDrives().Get(ctx, tt.driveObject.Name, metav1.GetOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			})
			if dErr != nil {
				t.Fatalf("Error while getting the drive object: %+v", dErr)
			}

			now := metav1.Now()
			newObj.ObjectMeta.DeletionTimestamp = &now
			if err := dl.Update(ctx, &tt.driveObject, newObj); err != nil && !tt.expectErr {
				t.Errorf("Error while invoking the update listener: %+v", err)
			}

			csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, newObj.Name, metav1.GetOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			})
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
