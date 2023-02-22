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

package volume

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KB = 1 << 10
	MB = KB << 10

	mb50  = 50 * MB
	mb100 = 100 * MB
	mb20  = 20 * MB
	mb30  = 30 * MB
)

func createFakeVolumeEventListener(nodeName string) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID: nodeName,
		safeUnmount: func(target string, force, detach, expire bool) error {
			return nil
		},
	}
}

func TestVolumeEventHandlerHandle(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName20MB := "test_volume_20MB"
	testVolumeName30MB := "test_volume_30MB"
	testDriveObject := &directcsi.DirectCSIDrive{
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
			NodeName:          "test-node",
			DriveStatus:       directcsi.DriveStatusInUse,
			FreeCapacity:      mb50,
			AllocatedCapacity: mb50,
			TotalCapacity:     mb100,
		},
	}
	testVolumeObjects := []runtime.Object{
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName20MB,
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      "test-node",
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
				NodeName:      "test-node",
				HostPath:      "hostpath",
				Drive:         testDriveName,
				TotalCapacity: mb30,
				StagingPath:   "/path/staging",
				ContainerPath: "/path/container",
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

	vl := createFakeVolumeEventListener("test-node")
	ctx := context.TODO()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDriveObject).DirectV1beta5().DirectCSIDrives())
	client.SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset(testVolumeObjects...).DirectV1beta5().DirectCSIVolumes())
	for _, testObj := range testVolumeObjects {
		var stagingUmountCalled, containerUmountCalled bool
		vl.safeUnmount = func(target string, force, detach, expire bool) error {
			if testObj.(*directcsi.DirectCSIVolume).Status.StagingPath == "" && testObj.(*directcsi.DirectCSIVolume).Status.ContainerPath == "" {
				return errors.New("umount should never be called for volumes with empty staging and container paths")
			}
			if target == testObj.(*directcsi.DirectCSIVolume).Status.StagingPath {
				stagingUmountCalled = true
			}
			if target == testObj.(*directcsi.DirectCSIVolume).Status.ContainerPath {
				containerUmountCalled = true
			}
			return nil
		}
		vObj, ok := testObj.(*directcsi.DirectCSIVolume)
		if !ok {
			continue
		}
		newObj, vErr := client.GetLatestDirectCSIVolumeInterface().Get(ctx, vObj.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()})
		if vErr != nil {
			t.Fatalf("Error while getting the volume object: %+v", vErr)
		}

		now := metav1.Now()
		newObj.DeletionTimestamp = &now

		_, vErr = client.GetLatestDirectCSIVolumeInterface().Update(
			ctx, newObj, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
		)
		if vErr != nil {
			t.Fatalf("Error while updating the volume object: %+v", vErr)
		}
		if err := vl.Handle(ctx, listener.Event{Type: listener.DeleteEvent, Object: newObj}); err != nil {
			t.Fatalf("Error while invoking the volume delete listener: %+v", err)
		}
		if newObj.Status.StagingPath != "" && !stagingUmountCalled {
			t.Error("staging path is not umounted")
		}
		if newObj.Status.ContainerPath != "" && !containerUmountCalled {
			t.Error("container path is not umounted")
		}
		updatedVolume, err := client.GetLatestDirectCSIVolumeInterface().Get(
			ctx, newObj.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
		)
		if err != nil {
			t.Fatalf("Error while getting the volume object: %+v", err)
		}
		if len(updatedVolume.GetFinalizers()) != 0 {
			t.Errorf("Volume finalizers are not empty: %v", updatedVolume.GetFinalizers())
		}
	}

	driveObj, dErr := client.GetLatestDirectCSIDriveInterface().Get(ctx, testDriveName, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
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

func TestAbnormalDeleteEventHandle(t *testing.T) {
	testVolumeObject := &directcsi.DirectCSIVolume{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-volume",
			Finalizers: []string{
				string(directcsi.DirectCSIVolumeFinalizerPVProtection),
				string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
			},
		},
		Status: directcsi.DirectCSIVolumeStatus{
			NodeName:      "test-node",
			HostPath:      "hostpath",
			Drive:         "test-drive",
			TotalCapacity: int64(100),
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
	}

	vl := createFakeVolumeEventListener("test-node")
	ctx := context.TODO()
	client.SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset(testVolumeObject).DirectV1beta5().DirectCSIVolumes())

	newObj, vErr := client.GetLatestDirectCSIVolumeInterface().Get(ctx, testVolumeObject.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()})
	if vErr != nil {
		t.Fatalf("Error while getting the volume object: %+v", vErr)
	}
	now := metav1.Now()
	newObj.DeletionTimestamp = &now
	_, vErr = client.GetLatestDirectCSIVolumeInterface().Update(
		ctx, newObj, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
	)
	if vErr != nil {
		t.Fatalf("Error while updating the volume object: %+v", vErr)
	}
	if err := vl.Handle(ctx, listener.Event{Type: listener.DeleteEvent, Object: newObj}); err == nil {
		t.Errorf("[%s] DeleteVolumeHandle expected to fail but succeeded", newObj.Name)
	}
}
