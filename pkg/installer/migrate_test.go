// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package installer

import (
	"context"
	"reflect"
	"testing"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	directv1beta5 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	legacyclientsetfake "github.com/minio/directpv/pkg/legacy/clientset/fake"
	"github.com/minio/directpv/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMigrateDrivesError(t *testing.T) {
	// invalid FilesystemUUID
	drive := &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "887e88cc-16ca-428f-9637-fde599b21b26"},
		Status:     directv1beta5.DirectCSIDriveStatus{DriveStatus: directv1beta5.DriveStatusReady},
	}

	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive))
	driveMap, legacyDriveErrors, driveErrors, err := migrateDrives(context.TODO(), false, nil)
	if len(driveMap) == 0 && len(legacyDriveErrors) == 0 && len(driveErrors) == 0 && err == nil {
		t.Fatalf("expected error, but succeeded\n")
	}

	drive = &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "7d552939-1daf-4887-8879-777c503cb1d7"},
		Status: directv1beta5.DirectCSIDriveStatus{
			DriveStatus:    directv1beta5.DriveStatusInUse,
			FilesystemUUID: "5e6849e1126441c18d51e3c568acc6fc",
		},
	}
	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive))
	driveMap, legacyDriveErrors, driveErrors, err = migrateDrives(context.TODO(), false, nil)
	if len(driveMap) == 0 && len(legacyDriveErrors) == 0 && len(driveErrors) == 0 && err == nil {
		t.Fatalf("expected error, but succeeded\n")
	}

	// duplicate FilesystemUUID
	drive1 := &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "887e88cc-16ca-428f-9637-fde599b21b26"},
		Status: directv1beta5.DirectCSIDriveStatus{
			DriveStatus:    directv1beta5.DriveStatusReady,
			FilesystemUUID: "3fb25851-18aa-48f2-8972-5d07c48154e5",
		},
	}
	drive2 := &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "7d552939-1daf-4887-8879-777c503cb1d7"},
		Status: directv1beta5.DirectCSIDriveStatus{
			DriveStatus:    directv1beta5.DriveStatusInUse,
			FilesystemUUID: "3fb25851-18aa-48f2-8972-5d07c48154e5",
		},
	}
	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive1, drive2))
	driveMap, legacyDriveErrors, driveErrors, err = migrateDrives(context.TODO(), false, nil)
	if len(driveMap) == 0 && len(legacyDriveErrors) == 0 && len(driveErrors) == 0 && err == nil {
		t.Fatalf("expected error, but succeeded\n")
	}
}

func TestMigrateNoDrives(t *testing.T) {
	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset())
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	_, legacyDriveErrors, driveErrors, err := migrateDrives(context.TODO(), false, nil)
	if len(legacyDriveErrors) != 0 || len(driveErrors) != 0 || err != nil {
		t.Fatalf("unexpected error; %v\n", err)
	}
	driveList, err := client.DriveClient().List(context.TODO(), metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error; %v\n", err)
	}
	if driveList != nil && len(driveList.Items) != 0 {
		t.Fatalf("expected: <empty>; got: %v\n", len(driveList.Items))
	}

	drive1 := &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "887e88cc-16ca-428f-9637-fde599b21b26"},
		Status:     directv1beta5.DirectCSIDriveStatus{DriveStatus: directv1beta5.DriveStatusAvailable},
	}
	drive2 := &directv1beta5.DirectCSIDrive{
		TypeMeta:   legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "7d552939-1daf-4887-8879-777c503cb1d7"},
		Status:     directv1beta5.DirectCSIDriveStatus{DriveStatus: directv1beta5.DriveStatusTerminating},
	}
	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive1, drive2))
	clientset = types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	_, legacyDriveErrors, driveErrors, err = migrateDrives(context.TODO(), false, nil)
	if len(legacyDriveErrors) != 0 || len(driveErrors) != 0 || err != nil {
		t.Fatalf("unexpected error; %v\n", err)
	}
	driveList, err = client.DriveClient().List(context.TODO(), metav1.ListOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error; %v\n", err)
	}
	if driveList != nil && len(driveList.Items) != 0 {
		t.Fatalf("expected: <empty>; got: %v\n", len(driveList.Items))
	}
}

func TestMigrateReadyDrive(t *testing.T) {
	drive := &directv1beta5.DirectCSIDrive{
		TypeMeta: legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:       "2bf15006-a710-4c5f-8678-e3c996baaf2f",
			Finalizers: []string{"direct.csi.min.io/data-protection"},
			Labels: map[string]string{
				"direct.csi.min.io/access-tier": "Unknown",
				"direct.csi.min.io/created-by":  "directcsi-driver",
				"direct.csi.min.io/node":        "c7",
				"direct.csi.min.io/path":        "vdb",
				"direct.csi.min.io/version":     "v1beta5",
			},
		},
		Status: directv1beta5.DirectCSIDriveStatus{
			AccessTier:        directv1beta5.AccessTierUnknown,
			AllocatedCapacity: 3616768,
			Conditions: []metav1.Condition{
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionOwned),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionMounted),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionFormatted),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonInitialized),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionInitialized),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonReady),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionReady),
				},
			},
			DriveStatus:    directv1beta5.DriveStatusReady,
			Filesystem:     "xfs",
			FilesystemUUID: "08450612-7ab3-40f9-ab83-38645fba6d29",
			FreeCapacity:   533254144,
			MajorNumber:    253,
			MinorNumber:    16,
			MountOptions:   []string{"noatime", "rw"},
			Mountpoint:     "/var/lib/direct-csi/mnt/08450612-7ab3-40f9-ab83-38645fba6d29",
			NodeName:       "c7",
			Path:           "/dev/vdb",
			PCIPath:        "pci-0000:08:00.0",
			RootPartition:  "vdb",
			Topology: map[string]string{
				"direct.csi.min.io/identity": "direct-csi-min-io",
				"direct.csi.min.io/node":     "c7",
				"direct.csi.min.io/rack":     "default",
				"direct.csi.min.io/region":   "default",
				"direct.csi.min.io/zone":     "default",
			},
			TotalCapacity: 536870912,
			UeventFSUUID:  "08450612-7ab3-40f9-ab83-38645fba6d29",
		},
	}

	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive))
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	driveMap, legacyDriveErrors, driveErrors, err := migrateDrives(context.TODO(), false, nil)
	if len(legacyDriveErrors) != 0 || len(driveErrors) != 0 || err != nil {
		t.Fatalf("unexpected error; %v, %v, %v\n", legacyDriveErrors, driveErrors, err)
	}
	if len(driveMap) == 0 {
		t.Fatalf("empty drive map\n")
	}

	result, err := client.DriveClient().Get(context.TODO(), "08450612-7ab3-40f9-ab83-38645fba6d29", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error; %v\n", err)
	}

	expectedResult := types.NewDrive(
		directpvtypes.DriveID("08450612-7ab3-40f9-ab83-38645fba6d29"),
		types.DriveStatus{
			TotalCapacity:     536870912,
			AllocatedCapacity: 3616768,
			FreeCapacity:      533254144,
			FSUUID:            "08450612-7ab3-40f9-ab83-38645fba6d29",
			Status:            directpvtypes.DriveStatusReady,
			Topology: map[string]string{
				"directpv.min.io/identity": "directpv-min-io",
				"directpv.min.io/node":     "c7",
				"directpv.min.io/rack":     "default",
				"directpv.min.io/region":   "default",
				"directpv.min.io/zone":     "default",
			},
		},
		directpvtypes.NodeID("c7"),
		directpvtypes.DriveName("vdb"),
		directpvtypes.AccessTierDefault,
	)
	expectedResult.SetMigratedLabel()
	expectedResult.Spec.Relabel = true

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v; got: %#+v\n", expectedResult, result)
	}
}

func TestMigrateInUseDrive(t *testing.T) {
	drive := &directv1beta5.DirectCSIDrive{
		TypeMeta: legacyclient.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: "08450612-7ab3-40f9-ab83-38645fba6d29",
			Finalizers: []string{
				"direct.csi.min.io/data-protection",
				"direct.csi.min.io.volume/pvc-c60680cc-c698-4dae-9f11-67611aeb563f",
				"direct.csi.min.io.volume/pvc-bfcbb9a7-1781-4c05-8da1-4d087688a213",
				"direct.csi.min.io.volume/pvc-4cf566ce-03cc-442a-b1ec-eb48897e3453",
				"direct.csi.min.io.volume/pvc-7b6b5cd2-f9b4-4958-b8db-e071b2d1c5a1",
				"direct.csi.min.io.volume/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5",
				"direct.csi.min.io.volume/pvc-1bace7a4-c575-429f-9e76-dcd3b14255c8",
				"direct.csi.min.io.volume/pvc-bff0997f-b442-4a19-89cb-eb43c132c207",
				"direct.csi.min.io.volume/pvc-aac3b633-9265-4288-9c97-4de0de36a546",
				"direct.csi.min.io.volume/pvc-d745d7fa-2b64-4dfb-aece-4b664f4db939",
				"direct.csi.min.io.volume/pvc-1b098369-faad-453b-80fe-d820e0f2da88",
				"direct.csi.min.io.volume/pvc-8f074641-da31-4867-8d66-c6f65cfd64c9",
				"direct.csi.min.io.volume/pvc-f1d4cdc5-0855-4c82-92f7-b62f90ac018e",
				"direct.csi.min.io.volume/pvc-6cf26e90-2b01-421d-9986-e97b2a30bd81",
				"direct.csi.min.io.volume/pvc-a7b6013e-9eb6-41f8-8d56-a94451f95587",
				"direct.csi.min.io.volume/pvc-10f64e68-5f02-4939-a941-7a6f24ad7dc5",
				"direct.csi.min.io.volume/pvc-6123a87f-8f4e-4e4f-991a-fdd23aadf158",
			},
			Labels: map[string]string{
				"direct.csi.min.io/access-tier": "Unknown",
				"direct.csi.min.io/created-by":  "directcsi-driver",
				"direct.csi.min.io/node":        "c7",
				"direct.csi.min.io/path":        "vdb",
				"direct.csi.min.io/version":     "v1beta5",
			},
		},
		Status: directv1beta5.DirectCSIDriveStatus{
			AccessTier:        directv1beta5.AccessTierUnknown,
			AllocatedCapacity: 272052224,
			Conditions: []metav1.Condition{
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionOwned),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionMounted),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonAdded),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionFormatted),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonInitialized),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionInitialized),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIDriveReasonReady),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIDriveConditionReady),
				},
			},
			DriveStatus:    directv1beta5.DriveStatusInUse,
			Filesystem:     "xfs",
			FilesystemUUID: "08450612-7ab3-40f9-ab83-38645fba6d29",
			FreeCapacity:   264818688,
			MajorNumber:    253,
			MinorNumber:    16,
			MountOptions:   []string{"noatime", "rw"},
			Mountpoint:     "/var/lib/direct-csi/mnt/08450612-7ab3-40f9-ab83-38645fba6d29",
			NodeName:       "c7",
			Path:           "/dev/vdb",
			PCIPath:        "pci-0000:08:00.0",
			RootPartition:  "vdb",
			Topology: map[string]string{
				"direct.csi.min.io/identity": "direct-csi-min-io",
				"direct.csi.min.io/node":     "c7",
				"direct.csi.min.io/rack":     "default",
				"direct.csi.min.io/region":   "default",
				"direct.csi.min.io/zone":     "default",
			},
			TotalCapacity: 536870912,
			UeventFSUUID:  "08450612-7ab3-40f9-ab83-38645fba6d29",
		},
	}

	legacyclient.SetDriveClient(legacyclientsetfake.NewSimpleClientset(drive))
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	driveMap, legacyDriveErrors, driveErrors, err := migrateDrives(context.TODO(), false, nil)
	if len(legacyDriveErrors) != 0 || len(driveErrors) != 0 || err != nil {
		t.Fatalf("unexpected error; %v, %v, %v\n", legacyDriveErrors, driveErrors, err)
	}
	if len(driveMap) == 0 {
		t.Fatalf("empty drive map\n")
	}

	result, err := client.DriveClient().Get(context.TODO(), "08450612-7ab3-40f9-ab83-38645fba6d29", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error; %v\n", err)
	}

	expectedResult := types.NewDrive(
		directpvtypes.DriveID("08450612-7ab3-40f9-ab83-38645fba6d29"),
		types.DriveStatus{
			TotalCapacity:     536870912,
			AllocatedCapacity: 272052224,
			FreeCapacity:      264818688,
			FSUUID:            "08450612-7ab3-40f9-ab83-38645fba6d29",
			Status:            directpvtypes.DriveStatusReady,
			Topology: map[string]string{
				"directpv.min.io/identity": "directpv-min-io",
				"directpv.min.io/node":     "c7",
				"directpv.min.io/rack":     "default",
				"directpv.min.io/region":   "default",
				"directpv.min.io/zone":     "default",
			},
		},
		directpvtypes.NodeID("c7"),
		directpvtypes.DriveName("vdb"),
		directpvtypes.AccessTierDefault,
	)
	expectedResult.SetMigratedLabel()
	expectedResult.Spec.Relabel = true
	expectedResult.AddVolumeFinalizer("pvc-c60680cc-c698-4dae-9f11-67611aeb563f")
	expectedResult.AddVolumeFinalizer("pvc-bfcbb9a7-1781-4c05-8da1-4d087688a213")
	expectedResult.AddVolumeFinalizer("pvc-4cf566ce-03cc-442a-b1ec-eb48897e3453")
	expectedResult.AddVolumeFinalizer("pvc-7b6b5cd2-f9b4-4958-b8db-e071b2d1c5a1")
	expectedResult.AddVolumeFinalizer("pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5")
	expectedResult.AddVolumeFinalizer("pvc-1bace7a4-c575-429f-9e76-dcd3b14255c8")
	expectedResult.AddVolumeFinalizer("pvc-bff0997f-b442-4a19-89cb-eb43c132c207")
	expectedResult.AddVolumeFinalizer("pvc-aac3b633-9265-4288-9c97-4de0de36a546")
	expectedResult.AddVolumeFinalizer("pvc-d745d7fa-2b64-4dfb-aece-4b664f4db939")
	expectedResult.AddVolumeFinalizer("pvc-1b098369-faad-453b-80fe-d820e0f2da88")
	expectedResult.AddVolumeFinalizer("pvc-8f074641-da31-4867-8d66-c6f65cfd64c9")
	expectedResult.AddVolumeFinalizer("pvc-f1d4cdc5-0855-4c82-92f7-b62f90ac018e")
	expectedResult.AddVolumeFinalizer("pvc-6cf26e90-2b01-421d-9986-e97b2a30bd81")
	expectedResult.AddVolumeFinalizer("pvc-a7b6013e-9eb6-41f8-8d56-a94451f95587")
	expectedResult.AddVolumeFinalizer("pvc-10f64e68-5f02-4939-a941-7a6f24ad7dc5")
	expectedResult.AddVolumeFinalizer("pvc-6123a87f-8f4e-4e4f-991a-fdd23aadf158")

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v; got: %#+v\n", expectedResult, result)
	}
}

func TestMigrateVolumes(t *testing.T) {
	volume := &directv1beta5.DirectCSIVolume{
		TypeMeta: legacyclient.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:       "pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5",
			Finalizers: []string{"direct.csi.min.io/pv-protection", "direct.csi.min.io/purge-protection"},
			Labels: map[string]string{
				"direct.csi.min.io/app":           "minio-example",
				"direct.csi.min.io/created-by":    "directcsi-controller",
				"direct.csi.min.io/drive":         "08450612-7ab3-40f9-ab83-38645fba6d29",
				"direct.csi.min.io/drive-path":    "vdb",
				"direct.csi.min.io/node":          "c7",
				"direct.csi.min.io/organization":  "minio",
				"direct.csi.min.io/pod.name":      "minio-1",
				"direct.csi.min.io/pod.namespace": "default",
				"direct.csi.min.io/tenant":        "tenant-1",
				"direct.csi.min.io/version":       "v1beta5",
			},
		},
		Status: directv1beta5.DirectCSIVolumeStatus{
			AvailableCapacity: 16777216,
			Conditions: []metav1.Condition{
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIVolumeReasonInUse),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIVolumeConditionStaged),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIVolumeReasonInUse),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIVolumeConditionPublished),
				},
				{
					LastTransitionTime: metav1.Now(),
					Reason:             string(directv1beta5.DirectCSIVolumeReasonReady),
					Status:             metav1.ConditionTrue,
					Type:               string(directv1beta5.DirectCSIVolumeConditionReady),
				},
			},
			ContainerPath: "/var/lib/kubelet/pods/52a3bbb9-30bd-429d-85b1-f1ada882e0ce/volumes/kubernetes.io~csi/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5/mount",
			Drive:         "08450612-7ab3-40f9-ab83-38645fba6d29",
			HostPath:      "/var/lib/direct-csi/mnt/08450612-7ab3-40f9-ab83-38645fba6d29/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5",
			NodeName:      "c7",
			StagingPath:   "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5/globalmount",
			TotalCapacity: 16777216,
		},
	}

	legacyclient.SetVolumeClient(legacyclientsetfake.NewSimpleClientset(volume))
	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	legacyVolumeErrors, volumeErrors, err := migrateVolumes(
		context.TODO(),
		map[string]string{"08450612-7ab3-40f9-ab83-38645fba6d29": "a9908089-96dd-4e8b-8f06-0c0b5e391f39"},
		false,
		nil,
	)
	if len(legacyVolumeErrors) != 0 || len(volumeErrors) != 0 || err != nil {
		t.Fatalf("unexpected error; %v, %v, %v\n", legacyVolumeErrors, volumeErrors, err)
	}

	result, err := client.VolumeClient().Get(context.TODO(), "pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error; %v\n", err)
	}

	expectedResult := types.NewVolume(
		"pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5",
		"a9908089-96dd-4e8b-8f06-0c0b5e391f39",
		directpvtypes.NodeID("c7"),
		directpvtypes.DriveID("a9908089-96dd-4e8b-8f06-0c0b5e391f39"),
		directpvtypes.DriveName("vdb"),
		16777216,
	)
	expectedResult.SetMigratedLabel()
	expectedResult.Status.DataPath = "/var/lib/direct-csi/mnt/08450612-7ab3-40f9-ab83-38645fba6d29/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5"
	expectedResult.Status.StagingTargetPath = "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5/globalmount"
	expectedResult.Status.TargetPath = "/var/lib/kubelet/pods/52a3bbb9-30bd-429d-85b1-f1ada882e0ce/volumes/kubernetes.io~csi/pvc-009bfc49-4a66-4055-9f19-bd039cc3b4f5/mount"
	expectedResult.SetPodName("minio-1")
	expectedResult.SetPodNS("default")
	expectedResult.Labels["directpv.min.io/app"] = "minio-example"
	expectedResult.Labels["directpv.min.io/organization"] = "minio"
	expectedResult.Labels["directpv.min.io/tenant"] = "tenant-1"
	expectedResult.Status.Status = directpvtypes.VolumeStatusReady

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected: %#+v; got: %#+v\n", expectedResult, result)
	}

	// no fsuuid found error
	legacyclient.SetVolumeClient(legacyclientsetfake.NewSimpleClientset(volume))
	legacyVolumeErrors, volumeErrors, err = migrateVolumes(context.TODO(), map[string]string{}, false, nil)
	if len(legacyVolumeErrors) == 0 && len(volumeErrors) == 0 && err == nil {
		t.Fatalf("expected error; but succeeded\n")
	}
}
