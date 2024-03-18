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
	"github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const MiB = 1024 * 1024

func createFakeVolumeEventListener(nodeID directpvtypes.NodeID) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID:            nodeID,
		unmount:           func(_ string) error { return nil },
		getDeviceByFSUUID: func(_ string) (string, error) { return "", nil },
		removeQuota:       func(_ context.Context, _, _, _ string) error { return nil },
	}
}

func TestVolumeEventHandlerHandle(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName20MB := "test_volume_20MB"
	testVolumeName30MB := "test_volume_30MB"
	testDriveObject := types.NewDrive(
		directpvtypes.DriveID(testDriveName),
		types.DriveStatus{
			TotalCapacity:     100 * MiB,
			FreeCapacity:      50 * MiB,
			AllocatedCapacity: 50 * MiB,
			Status:            directpvtypes.DriveStatusReady,
		},
		"test-node",
		"sda",
		directpvtypes.AccessTierDefault,
	)
	testDriveObject.AddVolumeFinalizer(testVolumeName20MB)
	testDriveObject.AddVolumeFinalizer(testVolumeName30MB)

	volume := types.NewVolume(
		testVolumeName30MB,
		"fsuuid1",
		"test-node",
		directpvtypes.DriveID(testDriveName),
		directpvtypes.DriveName(testDriveName),
		30*MiB,
	)
	volume.Status.StagingTargetPath = "/path/staging"
	volume.Status.TargetPath = "/path/target"
	testVolumeObjects := []runtime.Object{
		types.NewVolume(
			testVolumeName20MB,
			"fsuuid1",
			"test-node",
			directpvtypes.DriveID(testDriveName),
			directpvtypes.DriveName(testDriveName),
			20*MiB,
		),
		volume,
	}

	vl := createFakeVolumeEventListener("test-node")
	ctx := context.TODO()

	clientset1 := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testDriveObject))
	client.SetDriveInterface(clientset1.DirectpvLatest().DirectPVDrives())
	clientset2 := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVolumeObjects...))
	client.SetVolumeInterface(clientset2.DirectpvLatest().DirectPVVolumes())

	for _, testObj := range testVolumeObjects {
		var stagingUmountCalled, targetUmountCalled bool
		vl.unmount = func(target string) error {
			if testObj.(*types.Volume).Status.StagingTargetPath == "" && testObj.(*types.Volume).Status.TargetPath == "" {
				return errors.New("umount should never be called for volumes with empty staging and target paths")
			}
			if target == testObj.(*types.Volume).Status.StagingTargetPath {
				stagingUmountCalled = true
			}
			if target == testObj.(*types.Volume).Status.TargetPath {
				targetUmountCalled = true
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
		newObj.RemovePVProtection()

		_, vErr = client.VolumeClient().Update(
			ctx, newObj, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if vErr != nil {
			t.Fatalf("Error while updating the volume object: %+v", vErr)
		}
		if err := vl.Handle(ctx, controller.DeleteEvent, newObj); err != nil {
			t.Fatalf("Error while invoking the volume delete controller: %+v", err)
		}
		if newObj.Status.StagingTargetPath != "" && !stagingUmountCalled {
			t.Error("staging target path is not umounted")
		}
		if newObj.Status.TargetPath != "" && !targetUmountCalled {
			t.Error("target path is not umounted")
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

	if driveObj.GetVolumeCount() != 0 {
		t.Fatalf("Unexpected drive finalizers set after clean-up: %+v", driveObj.GetFinalizers())
	}
	if driveObj.Status.Status != directpvtypes.DriveStatusReady {
		t.Errorf("Unexpected drive status set. Expected: %s, Got: %s", directpvtypes.DriveStatusReady, driveObj.Status.Status)
	}
	if driveObj.Status.FreeCapacity != 100*MiB {
		t.Errorf("Unexpected free capacity set. Expected: %d, Got: %d", 100*MiB, driveObj.Status.FreeCapacity)
	}
	if driveObj.Status.AllocatedCapacity != 0 {
		t.Errorf("Unexpected allocated capacity set. Expected: 0, Got: %d", driveObj.Status.AllocatedCapacity)
	}
}

func TestAbnormalDeleteEventHandle(t *testing.T) {
	testVolumeObject := types.NewVolume("test-volume", "fsuuid1", "test-node", "test-drive", "test-drive", 100)
	testVolumeObject.Status.DataPath = "data/path"

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
	if err := vl.Handle(ctx, controller.DeleteEvent, newObj); err == nil {
		t.Errorf("[%s] DeleteVolumeHandle expected to fail but succeeded", newObj.Name)
	}
}

func TestSync(t *testing.T) {
	newDrive := func(name, driveName, volume string) *types.Drive {
		drive := types.NewDrive(
			directpvtypes.DriveID(name),
			types.DriveStatus{
				TotalCapacity: 100,
				Make:          "make",
			},
			directpvtypes.NodeID("nodeId"),
			directpvtypes.DriveName(driveName),
			directpvtypes.AccessTierDefault,
		)
		drive.AddVolumeFinalizer(volume)
		return drive
	}

	newVolume := func(name, driveID, driveName string) *types.Volume {
		volume := types.NewVolume(
			name,
			"fsuuid",
			"nodeId",
			directpvtypes.DriveID(driveID),
			directpvtypes.DriveName(driveName),
			100,
		)
		volume.Status.DataPath = "datapath"
		return volume
	}

	drive := newDrive("drive-1", "sda", "volume-1")
	volume := newVolume("volume-1", "drive-1", "sdb")
	objects := []runtime.Object{
		drive,
		volume,
	}
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(objects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	volume, err := client.VolumeClient().Get(context.TODO(), volume.Name, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		t.Fatalf("Volume (%s) not found; %v", volume.Name, err)
	}

	err = sync(context.TODO(), volume)
	if err != nil {
		t.Fatalf("unable to sync; %v", err)
	}

	volume, err = client.VolumeClient().Get(context.TODO(), volume.Name, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		t.Fatalf("Volume (%s) not found after sync; %v", volume.Name, err)
	}

	if volume.GetDriveName() != drive.GetDriveName() {
		t.Fatalf("expected drive name: %v; but got %v", drive.GetDriveName(), volume.GetDriveName())
	}
}
