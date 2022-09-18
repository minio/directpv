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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	testDriveObject := &types.Drive{
		TypeMeta: types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: testDriveName,
			Finalizers: []string{
				string(consts.DriveFinalizerDataProtection),
				consts.DriveFinalizerPrefix + testVolumeName20MB,
				consts.DriveFinalizerPrefix + testVolumeName30MB,
			},
		},
		Status: types.DriveStatus{
			NodeName:          "test-node",
			Status:            directpvtypes.DriveStatusOK,
			FreeCapacity:      mb50,
			AllocatedCapacity: mb50,
			TotalCapacity:     mb100,
		},
	}
	testVolumeObjects := []runtime.Object{
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName20MB,
				Finalizers: []string{
					string(consts.VolumeFinalizerPurgeProtection),
				},
			},
			Status: types.VolumeStatus{
				NodeName:      "test-node",
				HostPath:      "hostpath",
				DriveName:     testDriveName,
				TotalCapacity: mb20,
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeStaged),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string((directpvtypes.VolumeConditionReasonInUse)),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directpvtypes.VolumeConditionTypePublished),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directpvtypes.VolumeConditionReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directpvtypes.VolumeConditionTypeReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string((directpvtypes.VolumeConditionReasonReady)),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName30MB,
				Finalizers: []string{
					string(consts.VolumeFinalizerPurgeProtection),
				},
			},
			Status: types.VolumeStatus{
				NodeName:      "test-node",
				HostPath:      "hostpath",
				DriveName:     testDriveName,
				TotalCapacity: mb30,
				StagingPath:   "/path/staging",
				ContainerPath: "/path/container",
				Conditions: []metav1.Condition{
					{
						Type:               string(directpvtypes.VolumeConditionTypeStaged),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directpvtypes.VolumeConditionReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directpvtypes.VolumeConditionTypePublished),
						Status:             metav1.ConditionFalse,
						Message:            "",
						Reason:             string(directpvtypes.VolumeConditionReasonNotInUse),
						LastTransitionTime: metav1.Now(),
					},
					{
						Type:               string(directpvtypes.VolumeConditionTypeReady),
						Status:             metav1.ConditionTrue,
						Message:            "",
						Reason:             string((directpvtypes.VolumeConditionReasonReady)),
						LastTransitionTime: metav1.Now(),
					},
				},
			},
		},
	}

	vl := createFakeVolumeEventListener("test-node")
	ctx := context.TODO()

	clientset1 := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testDriveObject))
	client.SetDriveInterface(clientset1.DirectpvLatest().DirectPVDrives())
	clientset2 := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVolumeObjects...))
	client.SetVolumeInterface(clientset2.DirectpvLatest().DirectPVVolumes())

	for _, testObj := range testVolumeObjects {
		var stagingUmountCalled, containerUmountCalled bool
		vl.safeUnmount = func(target string, force, detach, expire bool) error {
			if testObj.(*types.Volume).Status.StagingPath == "" && testObj.(*types.Volume).Status.ContainerPath == "" {
				return errors.New("umount should never be called for volumes with empty staging and container paths")
			}
			if target == testObj.(*types.Volume).Status.StagingPath {
				stagingUmountCalled = true
			}
			if target == testObj.(*types.Volume).Status.ContainerPath {
				containerUmountCalled = true
			}
			return nil
		}
		vObj, ok := testObj.(*types.Volume)
		if !ok {
			continue
		}
		newObj, vErr := client.VolumeClient().Get(ctx, vObj.Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()})
		if vErr != nil {
			t.Fatalf("Error while getting the volume object: %+v", vErr)
		}

		now := metav1.Now()
		newObj.DeletionTimestamp = &now

		_, vErr = client.VolumeClient().Update(
			ctx, newObj, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if vErr != nil {
			t.Fatalf("Error while updating the volume object: %+v", vErr)
		}
		if err := vl.Handle(ctx, listener.EventArgs{Event: listener.DeleteEvent, Object: newObj}); err != nil {
			t.Fatalf("Error while invoking the volume delete listener: %+v", err)
		}
		if newObj.Status.StagingPath != "" && !stagingUmountCalled {
			t.Error("staging path is not umounted")
		}
		if newObj.Status.ContainerPath != "" && !containerUmountCalled {
			t.Error("container path is not umounted")
		}
		updatedVolume, err := client.VolumeClient().Get(
			ctx, newObj.Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if err != nil {
			t.Fatalf("Error while getting the volume object: %+v", err)
		}
		if len(updatedVolume.GetFinalizers()) != 0 {
			t.Errorf("Volume finalizers are not empty: %v", updatedVolume.GetFinalizers())
		}
	}

	driveObj, dErr := client.DriveClient().Get(ctx, testDriveName, metav1.GetOptions{TypeMeta: types.NewDriveTypeMeta()})
	if dErr != nil {
		t.Fatalf("Error while getting the drive object: %+v", dErr)
	}

	driveFinalizers := driveObj.GetFinalizers()
	if len(driveFinalizers) != 1 || driveFinalizers[0] != consts.DriveFinalizerDataProtection {
		t.Fatalf("Unexpected drive finalizers set after clean-up: %+v", driveFinalizers)
	}
	if driveObj.Status.Status != directpvtypes.DriveStatusOK {
		t.Errorf("Unexpected drive status set. Expected: %s, Got: %s", string(directpvtypes.DriveStatusOK), string(driveObj.Status.Status))
	}
	if driveObj.Status.FreeCapacity != mb100 {
		t.Errorf("Unexpected free capacity set. Expected: %d, Got: %d", mb100, driveObj.Status.FreeCapacity)
	}
	if driveObj.Status.AllocatedCapacity != 0 {
		t.Errorf("Unexpected allocated capacity set. Expected: 0, Got: %d", driveObj.Status.AllocatedCapacity)
	}
}

func TestAbnormalDeleteEventHandle(t *testing.T) {
	testVolumeObject := &types.Volume{
		TypeMeta: types.NewVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-volume",
			Finalizers: []string{
				string(consts.VolumeFinalizerPVProtection),
				string(consts.VolumeFinalizerPurgeProtection),
			},
		},
		Status: types.VolumeStatus{
			NodeName:      "test-node",
			HostPath:      "hostpath",
			DriveName:     "test-drive",
			TotalCapacity: int64(100),
			Conditions: []metav1.Condition{
				{
					Type:               string(directpvtypes.VolumeConditionTypeStaged),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string((directpvtypes.VolumeConditionReasonInUse)),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directpvtypes.VolumeConditionTypePublished),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directpvtypes.VolumeConditionReasonNotInUse),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directpvtypes.VolumeConditionTypeReady),
					Status:             metav1.ConditionTrue,
					Message:            "",
					Reason:             string((directpvtypes.VolumeConditionReasonReady)),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	vl := createFakeVolumeEventListener("test-node")
	ctx := context.TODO()

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVolumeObject))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	newObj, vErr := client.VolumeClient().Get(ctx, testVolumeObject.Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()})
	if vErr != nil {
		t.Fatalf("Error while getting the volume object: %+v", vErr)
	}
	now := metav1.Now()
	newObj.DeletionTimestamp = &now
	_, vErr = client.VolumeClient().Update(
		ctx, newObj, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)
	if vErr != nil {
		t.Fatalf("Error while updating the volume object: %+v", vErr)
	}
	if err := vl.Handle(ctx, listener.EventArgs{Event: listener.DeleteEvent, Object: newObj}); err == nil {
		t.Errorf("[%s] DeleteVolumeHandle expected to fail but succeeded", newObj.Name)
	}
}
