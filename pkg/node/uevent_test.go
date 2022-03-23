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

package node

import (
	"context"
	"sync"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createDriveEventHandler() *driveEventHandler {
	return &driveEventHandler{
		nodeID: "test-node",
		topology: map[string]string{
			string(utils.TopologyDriverIdentity): "identity",
			string(utils.TopologyDriverRack):     "rack",
			string(utils.TopologyDriverZone):     "zone",
			string(utils.TopologyDriverRegion):   "region",
			string(utils.TopologyDriverNode):     "test-node",
		},
	}
}

func TestAddHandler(t *testing.T) {
	testCases := []struct {
		device                     *sys.Device
		expectedDriveStatus        directcsi.DriveStatus
		expectedMountCondition     metav1.ConditionStatus
		expectedFormattedCondition metav1.ConditionStatus
	}{
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				FSUUID:            "fsuuid",
				FSType:            "xfs",
				UeventSerial:      "ueventserial",
				UeventFSUUID:      "ueventfsuuid",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
			expectedDriveStatus:        directcsi.DriveStatusAvailable,
			expectedMountCondition:     metav1.ConditionTrue,
			expectedFormattedCondition: metav1.ConditionTrue,
		},
		// drive not mounted
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				FSUUID:            "fsuuid",
				FSType:            "xfs",
				UeventSerial:      "ueventserial",
				UeventFSUUID:      "ueventfsuuid",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
			},
			expectedDriveStatus:        directcsi.DriveStatusAvailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionTrue,
		},
		// drive not formatted
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				UeventSerial:      "ueventserial",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
			},
			expectedDriveStatus:        directcsi.DriveStatusAvailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionFalse,
		},
		// Unavailable drive (less than minimum size)
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(sys.MinSupportedDeviceSize - 1),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				UeventSerial:      "ueventserial",
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
			},
			expectedDriveStatus:        directcsi.DriveStatusUnavailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionFalse,
		},
		// Unavailable drive (ReadOnly drive)
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				UeventSerial:      "ueventserial",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				ReadOnly:          true,
			},
			expectedDriveStatus:        directcsi.DriveStatusUnavailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionFalse,
		},
		// Unavailable drive (partitioned drive)
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				UeventSerial:      "ueventserial",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				Partitioned:       true,
			},
			expectedDriveStatus:        directcsi.DriveStatusUnavailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionFalse,
		},
		// Unavailable drive (slave drive)
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				UeventSerial:      "ueventserial",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				Master:            "dm-0",
			},
			expectedDriveStatus:        directcsi.DriveStatusUnavailable,
			expectedMountCondition:     metav1.ConditionFalse,
			expectedFormattedCondition: metav1.ConditionFalse,
		},
		// Unavailable drive (mounted outside)
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Serial:            "serial",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				FSUUID:            "fsuuid",
				FSType:            "xfs",
				UeventSerial:      "ueventserial",
				UeventFSUUID:      "ueventfsuuid",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/home"},
				FirstMountPoint:   "/home",
			},
			expectedDriveStatus:        directcsi.DriveStatusUnavailable,
			expectedMountCondition:     metav1.ConditionTrue,
			expectedFormattedCondition: metav1.ConditionTrue,
		},
	}

	ctx := context.TODO()
	handler := createDriveEventHandler()
	for i, testCase := range testCases {
		client.SetLatestDirectCSIDriveInterface(fakedirect.NewSimpleClientset().DirectV1beta4().DirectCSIDrives())

		if err := handler.Add(ctx, testCase.device); err != nil {
			t.Fatalf("case %d could not create drive: %v", i, err)
		}

		result, err := client.GetLatestDirectCSIDriveInterface().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("case %d could not list drives: %v", i, err)
		}

		if len(result.Items) != 1 {
			t.Fatalf("case %d found %d items after adding", i, len(result.Items))
		}

		drive := result.Items[0]
		if drive.Status.DriveStatus != testCase.expectedDriveStatus {
			t.Fatalf("case %d unexpected drive status. expected %v but got %v", i, directcsi.DriveStatusAvailable, drive.Status.DriveStatus)
		}
		if drive.Status.Path != "/dev/"+testCase.device.Name {
			t.Fatalf("case %d unexpected drive drive.Status.Path. expected %v but got %v", i, "/dev/"+testCase.device.Name, drive.Status.Path)
		}
		if drive.Status.RootPartition != testCase.device.Name {
			t.Fatalf("case %d unexpected drive drive.Status.RootPartition. expected %v but got %v", i, testCase.device.Name, drive.Status.RootPartition)
		}
		if drive.Status.MajorNumber != uint32(testCase.device.Major) {
			t.Fatalf("case %d unexpected drive drive.Status.MajorNumber. expected %v but got %v", i, testCase.device.Major, drive.Status.MajorNumber)
		}
		if drive.Status.MinorNumber != uint32(testCase.device.Minor) {
			t.Fatalf("case %d unexpected drive drive.Status.MinorNumber. expected %v but got %v", i, testCase.device.Minor, drive.Status.MinorNumber)
		}
		if drive.Status.WWID != testCase.device.WWID {
			t.Fatalf("case %d unexpected drive drive.Status.WWID. expected %v but got %v", i, testCase.device.WWID, drive.Status.WWID)
		}
		if drive.Status.ModelNumber != testCase.device.Model {
			t.Fatalf("case %d unexpected drive drive.Status.ModelNumber. expected %v but got %v", i, testCase.device.Model, drive.Status.ModelNumber)
		}
		if drive.Status.SerialNumber != testCase.device.Serial {
			t.Fatalf("case %d unexpected drive drive.Status.SerialNumber. expected %v but got %v", i, testCase.device.Serial, drive.Status.SerialNumber)
		}
		if drive.Status.Vendor != testCase.device.Vendor {
			t.Fatalf("case %d unexpected drive drive.Status.Vendor. expected %v but got %v", i, testCase.device.Vendor, drive.Status.Vendor)
		}
		if drive.Status.DMName != testCase.device.DMName {
			t.Fatalf("case %d unexpected drive drive.Status.DMName. expected %v but got %v", i, testCase.device.DMName, drive.Status.DMName)
		}
		if drive.Status.DMUUID != testCase.device.DMUUID {
			t.Fatalf("case %d unexpected drive drive.Status.DMUUID. expected %v but got %v", i, testCase.device.DMUUID, drive.Status.DMUUID)
		}
		if drive.Status.MDUUID != testCase.device.MDUUID {
			t.Fatalf("case %d unexpected drive drive.Status.MDUUID. expected %v but got %v", i, testCase.device.MDUUID, drive.Status.MDUUID)
		}
		if drive.Status.PartTableUUID != testCase.device.PTUUID {
			t.Fatalf("case %d unexpected drive drive.Status.PartTableUUID. expected %v but got %v", i, testCase.device.PTUUID, drive.Status.PartTableUUID)
		}
		if drive.Status.PartTableType != testCase.device.PTType {
			t.Fatalf("case %d unexpected drive drive.Status.PartTableType. expected %v but got %v", i, testCase.device.PTType, drive.Status.PartTableType)
		}
		if drive.Status.PartitionUUID != testCase.device.PartUUID {
			t.Fatalf("case %d unexpected drive drive.Status.PartitionUUID. expected %v but got %v", i, testCase.device.PartUUID, drive.Status.PartitionUUID)
		}
		if drive.Status.FilesystemUUID != testCase.device.FSUUID {
			t.Fatalf("case %d unexpected drive drive.Status.FilesystemUUID. expected %v but got %v", i, testCase.device.FSUUID, drive.Status.FilesystemUUID)
		}
		if drive.Status.Filesystem != testCase.device.FSType {
			t.Fatalf("case %d unexpected drive drive.Status.Filesystem. expected %v but got %v", i, testCase.device.FSType, drive.Status.Filesystem)
		}
		if drive.Status.UeventSerial != testCase.device.UeventSerial {
			t.Fatalf("case %d unexpected drive drive.Status.UeventSerial. expected %v but got %v", i, testCase.device.UeventSerial, drive.Status.UeventSerial)
		}
		if drive.Status.UeventFSUUID != testCase.device.UeventFSUUID {
			t.Fatalf("case %d unexpected drive drive.Status.UeventFSUUID. expected %v but got %v", i, testCase.device.UeventFSUUID, drive.Status.UeventFSUUID)
		}
		if drive.Status.TotalCapacity != int64(testCase.device.Size) {
			t.Fatalf("case %d unexpected drive drive.Status.TotalCapacity. expected %v but got %v", i, int64(testCase.device.Size), drive.Status.TotalCapacity)
		}
		if drive.Status.FreeCapacity != int64(testCase.device.FreeCapacity) {
			t.Fatalf("case %d unexpected drive drive.Status.FreeCapacity. expected %v but got %v", i, int64(testCase.device.FreeCapacity), drive.Status.FreeCapacity)
		}
		if drive.Status.AllocatedCapacity != int64(testCase.device.Size-testCase.device.FreeCapacity) {
			t.Fatalf("case %d unexpected drive drive.Status.AllocatedCapacity. expected %v but got %v", i, int64(testCase.device.Size-testCase.device.FreeCapacity), drive.Status.AllocatedCapacity)
		}
		if drive.Status.Mountpoint != testCase.device.FirstMountPoint {
			t.Fatalf("case %d unexpected drive drive.Status.Mountpoint. expected %v but got %v", i, testCase.device.FirstMountPoint, drive.Status.Mountpoint)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionMounted),
			testCase.expectedMountCondition,
		) {
			t.Fatalf("case %d unexpected mount condition status. expected %v", i, testCase.expectedMountCondition)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionFormatted),
			testCase.expectedFormattedCondition,
		) {
			t.Fatalf("case %d unexpected mount condition status. expected %v", i, testCase.expectedFormattedCondition)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionOwned),
			metav1.ConditionFalse,
		) {
			t.Fatalf("case %d found owned condition to be true", i)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionInitialized),
			metav1.ConditionTrue,
		) {
			t.Fatalf("case %d found initialized condition to be false", i)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionReady),
			metav1.ConditionTrue,
		) {
			t.Fatalf("case %d found Ready condition to be false", i)
		}
	}
}

func TestAddHandlerWithRace(t *testing.T) {
	devices := []*sys.Device{
		{
			Name:              "name",
			Major:             200,
			Minor:             2,
			Size:              uint64(5368709120),
			WWID:              "wwid",
			Model:             "model",
			Serial:            "serial",
			Vendor:            "vendor",
			DMName:            "dmname",
			DMUUID:            "dmuuid",
			MDUUID:            "mduuid",
			PTUUID:            "ptuuid",
			PTType:            "gpt",
			PartUUID:          "partuuid",
			FSUUID:            "fsuuid",
			FSType:            "xfs",
			UeventSerial:      "ueventserial",
			UeventFSUUID:      "ueventfsuuid",
			TotalCapacity:     uint64(5368709120),
			FreeCapacity:      uint64(5368709120),
			LogicalBlockSize:  uint64(512),
			PhysicalBlockSize: uint64(512),
			MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
			FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
		},
		{
			Name:              "name",
			Major:             200,
			Minor:             2,
			Size:              uint64(5368709120),
			WWID:              "wwid",
			Model:             "model",
			Serial:            "serial",
			Vendor:            "vendor",
			DMName:            "dmname",
			DMUUID:            "dmuuid",
			MDUUID:            "mduuid",
			PTUUID:            "ptuuid",
			PTType:            "gpt",
			PartUUID:          "partuuid",
			FSUUID:            "fsuuid",
			FSType:            "xfs",
			UeventSerial:      "ueventserial",
			UeventFSUUID:      "ueventfsuuid",
			TotalCapacity:     uint64(5368709120),
			FreeCapacity:      uint64(5368709120),
			LogicalBlockSize:  uint64(512),
			PhysicalBlockSize: uint64(512),
			MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
			FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
		},
		{
			Name:              "name",
			Major:             200,
			Minor:             2,
			Size:              uint64(5368709120),
			WWID:              "wwid",
			Model:             "model",
			Serial:            "serial",
			Vendor:            "vendor",
			DMName:            "dmname",
			DMUUID:            "dmuuid",
			MDUUID:            "mduuid",
			PTUUID:            "ptuuid",
			PTType:            "gpt",
			PartUUID:          "partuuid",
			FSUUID:            "fsuuid",
			FSType:            "xfs",
			UeventSerial:      "ueventserial",
			UeventFSUUID:      "ueventfsuuid",
			TotalCapacity:     uint64(5368709120),
			FreeCapacity:      uint64(5368709120),
			LogicalBlockSize:  uint64(512),
			PhysicalBlockSize: uint64(512),
			MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
			FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
		},
	}

	var wg sync.WaitGroup
	client.SetLatestDirectCSIDriveInterface(fakedirect.NewSimpleClientset().DirectV1beta4().DirectCSIDrives())
	ctx := context.TODO()
	handler := createDriveEventHandler()
	for _, device := range devices {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := handler.Add(ctx, device); err != nil {
				t.Fatalf("could not create drive: %v", err)
			}
		}()
	}
	wg.Wait()

	result, err := client.GetLatestDirectCSIDriveInterface().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("could not list drives: %v", err)
	}

	if len(result.Items) != 1 {
		t.Error("duplicate drives found")
	}
}

func TestUpdateHandler(t *testing.T) {
	testCases := []struct {
		device                 *sys.Device
		drive                  *directcsi.DirectCSIDrive
		expectedReadyCondition metav1.ConditionStatus
	}{
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-driv-1",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedReadyCondition: metav1.ConditionTrue,
		},
		// fill missing hwinfo
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					RootPartition:     "sdb1",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					DMName:            "vg0-lv0",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedReadyCondition: metav1.ConditionTrue,
		},
		// error scenario
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "e79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "e79dff9e-2884-46f2-8919-dada2eecb12d",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					RootPartition:     "sdb1",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					DMName:            "vg0-lv0",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusReady,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedReadyCondition: metav1.ConditionFalse,
		},
	}

	ctx := context.TODO()
	handler := createDriveEventHandler()
	for i, testCase := range testCases {
		client.SetLatestDirectCSIDriveInterface(fakedirect.NewSimpleClientset(testCase.drive).DirectV1beta4().DirectCSIDrives())

		if err := handler.Update(ctx, testCase.device, testCase.drive); err != nil {
			t.Fatalf("case %d could not create drive: %v", i, err)
		}

		drive, err := client.GetLatestDirectCSIDriveInterface().Get(ctx, testCase.drive.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("case %d could not get drive: %v", i, err)
		}

		if drive.Status.Path != "/dev/"+testCase.device.Name {
			t.Fatalf("case %d unexpected drive drive.Status.Path. expected %v but got %v", i, "/dev/"+testCase.device.Name, drive.Status.Path)
		}
		if drive.Status.RootPartition != testCase.device.Name {
			t.Fatalf("case %d unexpected drive drive.Status.RootPartition. expected %v but got %v", i, testCase.device.Name, drive.Status.RootPartition)
		}
		if drive.Status.MajorNumber != uint32(testCase.device.Major) {
			t.Fatalf("case %d unexpected drive drive.Status.MajorNumber. expected %v but got %v", i, testCase.device.Major, drive.Status.MajorNumber)
		}
		if drive.Status.MinorNumber != uint32(testCase.device.Minor) {
			t.Fatalf("case %d unexpected drive drive.Status.MinorNumber. expected %v but got %v", i, testCase.device.Minor, drive.Status.MinorNumber)
		}
		if drive.Status.WWID != testCase.device.WWID {
			t.Fatalf("case %d unexpected drive drive.Status.WWID. expected %v but got %v", i, testCase.device.WWID, drive.Status.WWID)
		}
		if drive.Status.ModelNumber != testCase.device.Model {
			t.Fatalf("case %d unexpected drive drive.Status.ModelNumber. expected %v but got %v", i, testCase.device.Model, drive.Status.ModelNumber)
		}
		if drive.Status.SerialNumber != testCase.device.Serial {
			t.Fatalf("case %d unexpected drive drive.Status.SerialNumber. expected %v but got %v", i, testCase.device.Serial, drive.Status.SerialNumber)
		}
		if drive.Status.Vendor != testCase.device.Vendor {
			t.Fatalf("case %d unexpected drive drive.Status.Vendor. expected %v but got %v", i, testCase.device.Vendor, drive.Status.Vendor)
		}
		if drive.Status.DMName != testCase.device.DMName {
			t.Fatalf("case %d unexpected drive drive.Status.DMName. expected %v but got %v", i, testCase.device.DMName, drive.Status.DMName)
		}
		if drive.Status.DMUUID != testCase.device.DMUUID {
			t.Fatalf("case %d unexpected drive drive.Status.DMUUID. expected %v but got %v", i, testCase.device.DMUUID, drive.Status.DMUUID)
		}
		if drive.Status.MDUUID != testCase.device.MDUUID {
			t.Fatalf("case %d unexpected drive drive.Status.MDUUID. expected %v but got %v", i, testCase.device.MDUUID, drive.Status.MDUUID)
		}
		if drive.Status.PartTableUUID != testCase.device.PTUUID {
			t.Fatalf("case %d unexpected drive drive.Status.PartTableUUID. expected %v but got %v", i, testCase.device.PTUUID, drive.Status.PartTableUUID)
		}
		if drive.Status.PartTableType != testCase.device.PTType {
			t.Fatalf("case %d unexpected drive drive.Status.PartTableType. expected %v but got %v", i, testCase.device.PTType, drive.Status.PartTableType)
		}
		if drive.Status.PartitionUUID != testCase.device.PartUUID {
			t.Fatalf("case %d unexpected drive drive.Status.PartitionUUID. expected %v but got %v", i, testCase.device.PartUUID, drive.Status.PartitionUUID)
		}
		if drive.Status.FilesystemUUID != testCase.device.FSUUID {
			t.Fatalf("case %d unexpected drive drive.Status.FilesystemUUID. expected %v but got %v", i, testCase.device.FSUUID, drive.Status.FilesystemUUID)
		}
		if drive.Status.Filesystem != testCase.device.FSType {
			t.Fatalf("case %d unexpected drive drive.Status.Filesystem. expected %v but got %v", i, testCase.device.FSType, drive.Status.Filesystem)
		}
		if drive.Status.UeventSerial != testCase.device.UeventSerial {
			t.Fatalf("case %d unexpected drive drive.Status.UeventSerial. expected %v but got %v", i, testCase.device.UeventSerial, drive.Status.UeventSerial)
		}
		if drive.Status.UeventFSUUID != testCase.device.UeventFSUUID {
			t.Fatalf("case %d unexpected drive drive.Status.UeventFSUUID. expected %v but got %v", i, testCase.device.UeventFSUUID, drive.Status.UeventFSUUID)
		}
		if drive.Status.TotalCapacity != int64(testCase.device.Size) {
			t.Fatalf("case %d unexpected drive drive.Status.TotalCapacity. expected %v but got %v", i, int64(testCase.device.Size), drive.Status.TotalCapacity)
		}
		if drive.Status.FreeCapacity != int64(testCase.device.FreeCapacity) {
			t.Fatalf("case %d unexpected drive drive.Status.FreeCapacity. expected %v but got %v", i, int64(testCase.device.FreeCapacity), drive.Status.FreeCapacity)
		}
		if drive.Status.AllocatedCapacity != int64(testCase.device.Size-testCase.device.FreeCapacity) {
			t.Fatalf("case %d unexpected drive drive.Status.AllocatedCapacity. expected %v but got %v", i, int64(testCase.device.Size-testCase.device.FreeCapacity), drive.Status.AllocatedCapacity)
		}
		if drive.Status.Mountpoint != testCase.device.FirstMountPoint {
			t.Fatalf("case %d unexpected drive drive.Status.Mountpoint. expected %v but got %v", i, testCase.device.FirstMountPoint, drive.Status.Mountpoint)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions,
			string(directcsi.DirectCSIDriveConditionReady),
			testCase.expectedReadyCondition,
		) {
			t.Fatalf("case %d found NotReady condition to be wrong", i)
		}
	}
}
