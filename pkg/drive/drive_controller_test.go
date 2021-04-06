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
	"testing"

	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNodeID = "test-node"
)

func createFakeDriveListener() *DirectCSIDriveListener {
	utils.SetFake()

	fakeKubeClnt := utils.GetKubeClient()
	fakeDirectCSIClnt := utils.GetDirectClientset()
	mounter := GetDriveMounter()
	formatter := GetDriveFormatter()
	statter := GetDriveStatter()

	return &DirectCSIDriveListener{
		kubeClient:      fakeKubeClnt,
		directcsiClient: fakeDirectCSIClnt,
		nodeID:          testNodeID,
		CRDVersion:      "direct.csi.min.io/v1beta1",
		mounter:         mounter,
		formatter:       formatter,
		statter:         statter,
	}
}

func TestAddAndDeleteDriveNoOp(t *testing.T) {
	dl := createFakeDriveListener()
	b := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(dl.CRDVersion),
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

// Creates a loop back device and tears it down after testing
// Sets the requested format in the Spec and checks if desired results are seen.
func TestDriveFormat(t *testing.T) {

	driveName := "test_drive"
	mountPath := filepath.Join(sys.MountRoot, driveName)

	oldObj := directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta("direct.csi.min.io/v1beta1"),
		ObjectMeta: metav1.ObjectMeta{
			Name: driveName,
		},
		Spec: directcsi.DirectCSIDriveSpec{
			DirectCSIOwned: false,
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:    testNodeID,
			DriveStatus: directcsi.DriveStatusAvailable,
			Path:        "/drive/path",
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
					Status:             metav1.ConditionFalse,
					Message:            "initialized",
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	ctx := context.TODO()

	// Step 1: Construct fake drive listener
	dl := createFakeDriveListener()
	dl.directcsiClient = fakedirect.NewSimpleClientset(&oldObj)

	// Step 2: Get the object
	directCSIClient := dl.directcsiClient.DirectV1beta1()
	newObj, dErr := directCSIClient.DirectCSIDrives().Get(ctx, oldObj.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta("direct.csi.min.io/v1beta1"),
	})

	// Step 3: Set RequestedFormat to enable formatting
	newObj.Spec.DirectCSIOwned = true
	newObj.Spec.RequestedFormat = &directcsi.RequestedFormat{
		Force:      true,
		Filesystem: string(sys.FSTypeXFS),
	}

	// Step 4: Execute the Update hook
	if err := dl.Update(ctx, &oldObj, newObj); err != nil {
		t.Errorf("Error while update: %+v", err)
	}

	// Step 5: Get the latest version of the object
	csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, newObj.Name, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta("direct.csi.min.io/v1beta1"),
	})
	if dErr != nil {
		t.Errorf("Error while update: %+v", dErr)
	}

	// Step 6: Check if the Status fields are updated
	if csiDrive.Status.DriveStatus != directcsi.DriveStatusReady {
		t.Errorf("Drive is not in 'ready' state after formatting. Current status: %s", csiDrive.Status.DriveStatus)
	}
	if csiDrive.Status.Mountpoint != mountPath {
		t.Errorf("Drive mountpath invalid: %s", csiDrive.Status.Mountpoint)
	}
	if csiDrive.Status.Filesystem != string(sys.FSTypeXFS) {
		t.Errorf("Invalid filesystem after formatting: %s", string(csiDrive.Status.Filesystem))
	}

	// Step 7: Check if the expected conditions are set
	if !utils.IsCondition(csiDrive.Status.Conditions, string(directcsi.DirectCSIDriveConditionOwned), metav1.ConditionTrue, string(directcsi.DirectCSIDriveReasonAdded), "") {
		t.Errorf("unexpected status.condition for %s = %v", string(directcsi.DirectCSIDriveConditionOwned), csiDrive.Status.Conditions)
	}
	if !utils.IsCondition(csiDrive.Status.Conditions, string(directcsi.DirectCSIDriveConditionMounted), metav1.ConditionTrue, string(directcsi.DirectCSIDriveReasonAdded), "mounted") {
		t.Errorf("unexpected status.condition for %s = %v", string(directcsi.DirectCSIDriveConditionMounted), csiDrive.Status.Conditions)
	}
	if !utils.IsCondition(csiDrive.Status.Conditions, string(directcsi.DirectCSIDriveConditionFormatted), metav1.ConditionTrue, string(directcsi.DirectCSIDriveReasonAdded), "formatted to xfs") {
		t.Errorf("unexpected status.condition for %s = %v", string(directcsi.DirectCSIDriveConditionFormatted), csiDrive.Status.Conditions)
	}
}
