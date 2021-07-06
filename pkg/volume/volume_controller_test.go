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

package volume

import (
	"context"
	"testing"

	"github.com/minio/direct-csi/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KB = 1 << 10
	MB = KB << 10

	mb50  = 50 * MB
	mb100 = 100 * MB
	mb20  = 20 * MB
	mb30  = 30 * MB

	testNodeName = "test-node"
)

func createFakeVolumeListener() *DirectCSIVolumeListener {
	utils.SetFake()
	fakeKubeClnt := utils.GetKubeClient()
	fakeDirectCSIClnt := utils.GetDirectClientset()
	return &DirectCSIVolumeListener{
		kubeClient:      fakeKubeClnt,
		directcsiClient: fakeDirectCSIClnt,
		nodeID:          testNodeName,
	}
}
func TestUpdateVolumeDelete(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName20MB := "test_volume_20MB"
	testVolumeName30MB := "test_volume_30MB"
	testObjects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testDriveName,
				Finalizers: []string{
					string(directcsi.DirectCSIDriveFinalizerDataProtection),
					directcsi.DirectCSIDriveFinalizerPrefix + testVolumeName20MB,
					directcsi.DirectCSIDriveFinalizerPrefix + testVolumeName30MB,
				},
			},
			Status: directcsi.DirectCSIDriveStatus{
				NodeName:          testNodeName,
				DriveStatus:       directcsi.DriveStatusInUse,
				FreeCapacity:      mb50,
				AllocatedCapacity: mb50,
				TotalCapacity:     mb100,
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName20MB,
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      testNodeName,
				HostPath:      "hostpath",
				Drive:         testDriveName,
				TotalCapacity: mb20,
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
				Name: testVolumeName30MB,
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      testNodeName,
				HostPath:      "hostpath",
				Drive:         testDriveName,
				TotalCapacity: mb30,
				Conditions: []metav1.Condition{
					{
						Type:               string(directcsi.DirectCSIVolumeConditionStaged),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
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
	}

	ctx := context.TODO()
	vl := createFakeVolumeListener()
	vl.directcsiClient = fakedirect.NewSimpleClientset(testObjects...)
	directCSIClient := vl.directcsiClient.DirectV1beta2()

	for _, testObj := range testObjects {
		vObj, ok := testObj.(*directcsi.DirectCSIVolume)
		if !ok {
			continue
		}
		newObj, vErr := directCSIClient.DirectCSIVolumes().Get(ctx, vObj.Name, metav1.GetOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		})
		if vErr != nil {
			t.Fatalf("Error while getting the drive object: %+v", vErr)
		}

		now := metav1.Now()
		newObj.ObjectMeta.DeletionTimestamp = &now

		if err := vl.Update(ctx, vObj, newObj); err != nil {
			t.Fatalf("Error while invoking the volume update listener: %+v", err)
		}
		if len(newObj.GetFinalizers()) != 0 {
			t.Errorf("Volume finalizers are not empty: %v", newObj.GetFinalizers())
		}
	}

	driveObj, dErr := directCSIClient.DirectCSIDrives().Get(ctx, testDriveName, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if dErr != nil {
		t.Fatalf("Error while getting the drive object: %+v", dErr)
	}

	driveFinalizers := driveObj.GetFinalizers()
	if len(driveFinalizers) != 1 || driveFinalizers[0] != directcsi.DirectCSIDriveFinalizerDataProtection {
		t.Fatalf("Unexpected drive finalizers set after clean-up: %+v", driveFinalizers)
	}
	if driveObj.Status.DriveStatus != directcsi.DriveStatusReady {
		t.Errorf("Unexpected drive status set. Expected: %s, Got: %s", string(directcsi.DriveStatusReady), string(driveObj.Status.DriveStatus))
	}
	if driveObj.Status.FreeCapacity != mb100 {
		t.Errorf("Unexpected free capacity set. Expected: %d, Got: %d", mb100, driveObj.Status.FreeCapacity)
	}
	if driveObj.Status.AllocatedCapacity != 0 {
		t.Errorf("Unexpected allocated capacity set. Expected: 0, Got: %d", driveObj.Status.AllocatedCapacity)
	}
}

func TestAddAndDeleteVolumeNoOp(t *testing.T) {
	vl := createFakeVolumeListener()
	b := directcsi.DirectCSIVolume{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test_volume",
		},
		Status: directcsi.DirectCSIVolumeStatus{},
	}
	ctx := context.TODO()
	err := vl.Add(ctx, &b)
	if err != nil {
		t.Errorf("Error returned [Add]: %+v", err)
	}

	err = vl.Delete(ctx, &b)
	if err != nil {
		t.Errorf("Error returned [Delete]: %+v", err)
	}

}
