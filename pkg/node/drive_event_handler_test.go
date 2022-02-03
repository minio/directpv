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
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
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
		createDevice: func(event map[string]string) (device *sys.Device, err error) {
			return &sys.Device{}, nil
		},
	}
}

func TestAddHandler(t *testing.T) {
	device := &sys.Device{
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
		Parent:            "parent",
		TotalCapacity:     uint64(5368709120),
		FreeCapacity:      uint64(5368709120),
		LogicalBlockSize:  uint64(512),
		PhysicalBlockSize: uint64(512),
		MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
		FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
	}

	client.SetLatestDirectCSIDriveInterface(fakedirect.NewSimpleClientset().DirectV1beta3().DirectCSIDrives())
	ctx := context.TODO()
	handler := createDriveEventHandler()
	if err := handler.add(ctx, device); err != nil {
		t.Fatalf("could not create drive: %v", err)
	}

	result, err := client.GetLatestDirectCSIDriveInterface().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("could not list drives: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("found %d items after adding", len(result.Items))
	}

	drive := result.Items[0]

	if drive.Status.DriveStatus != directcsi.DriveStatusAvailable {
		t.Fatalf("unexpected drive status. expected %v but got %v", directcsi.DriveStatusAvailable, drive.Status.DriveStatus)
	}
	if drive.Status.Path != "/dev/"+device.Name {
		t.Fatalf("unexpected drive drive.Status.Path. expected %v but got %v", "/dev/"+device.Name, drive.Status.Path)
	}
	if drive.Status.RootPartition != device.Name {
		t.Fatalf("unexpected drive drive.Status.RootPartition. expected %v but got %v", device.Name, drive.Status.RootPartition)
	}
	if drive.Status.MajorNumber != uint32(device.Major) {
		t.Fatalf("unexpected drive drive.Status.MajorNumber. expected %v but got %v", device.Major, drive.Status.MajorNumber)
	}
	if drive.Status.MinorNumber != uint32(device.Minor) {
		t.Fatalf("unexpected drive drive.Status.MinorNumber. expected %v but got %v", device.Minor, drive.Status.MinorNumber)
	}
	if drive.Status.WWID != device.WWID {
		t.Fatalf("unexpected drive drive.Status.WWID. expected %v but got %v", device.WWID, drive.Status.WWID)
	}
	if drive.Status.ModelNumber != device.Model {
		t.Fatalf("unexpected drive drive.Status.ModelNumber. expected %v but got %v", device.Model, drive.Status.ModelNumber)
	}
	if drive.Status.SerialNumber != device.Serial {
		t.Fatalf("unexpected drive drive.Status.SerialNumber. expected %v but got %v", device.Serial, drive.Status.SerialNumber)
	}
	if drive.Status.Vendor != device.Vendor {
		t.Fatalf("unexpected drive drive.Status.Vendor. expected %v but got %v", device.Vendor, drive.Status.Vendor)
	}
	if drive.Status.DMName != device.DMName {
		t.Fatalf("unexpected drive drive.Status.DMName. expected %v but got %v", device.DMName, drive.Status.DMName)
	}
	if drive.Status.DMUUID != device.DMUUID {
		t.Fatalf("unexpected drive drive.Status.DMUUID. expected %v but got %v", device.DMUUID, drive.Status.DMUUID)
	}
	if drive.Status.MDUUID != device.MDUUID {
		t.Fatalf("unexpected drive drive.Status.MDUUID. expected %v but got %v", device.MDUUID, drive.Status.MDUUID)
	}
	if drive.Status.PartTableUUID != device.PTUUID {
		t.Fatalf("unexpected drive drive.Status.PartTableUUID. expected %v but got %v", device.PTUUID, drive.Status.PartTableUUID)
	}
	if drive.Status.PartTableType != device.PTType {
		t.Fatalf("unexpected drive drive.Status.PartTableType. expected %v but got %v", device.PTType, drive.Status.PartTableType)
	}
	if drive.Status.PartitionUUID != device.PartUUID {
		t.Fatalf("unexpected drive drive.Status.PartitionUUID. expected %v but got %v", device.PartUUID, drive.Status.PartitionUUID)
	}
	if drive.Status.FilesystemUUID != device.FSUUID {
		t.Fatalf("unexpected drive drive.Status.FilesystemUUID. expected %v but got %v", device.FSUUID, drive.Status.FilesystemUUID)
	}
	if drive.Status.Filesystem != device.FSType {
		t.Fatalf("unexpected drive drive.Status.Filesystem. expected %v but got %v", device.FSType, drive.Status.Filesystem)
	}
	if drive.Status.UeventSerial != device.UeventSerial {
		t.Fatalf("unexpected drive drive.Status.UeventSerial. expected %v but got %v", device.UeventSerial, drive.Status.UeventSerial)
	}
	if drive.Status.UeventFSUUID != device.UeventFSUUID {
		t.Fatalf("unexpected drive drive.Status.UeventFSUUID. expected %v but got %v", device.UeventFSUUID, drive.Status.UeventFSUUID)
	}
	if drive.Status.TotalCapacity != int64(device.Size) {
		t.Fatalf("unexpected drive drive.Status.TotalCapacity. expected %v but got %v", int64(device.Size), drive.Status.TotalCapacity)
	}
	if drive.Status.FreeCapacity != int64(device.FreeCapacity) {
		t.Fatalf("unexpected drive drive.Status.FreeCapacity. expected %v but got %v", int64(device.FreeCapacity), drive.Status.FreeCapacity)
	}
	if drive.Status.AllocatedCapacity != int64(device.Size-device.FreeCapacity) {
		t.Fatalf("unexpected drive drive.Status.AllocatedCapacity. expected %v but got %v", int64(device.Size-device.FreeCapacity), drive.Status.AllocatedCapacity)
	}
	if drive.Status.Mountpoint != device.FirstMountPoint {
		t.Fatalf("unexpected drive drive.Status.Mountpoint. expected %v but got %v", device.FirstMountPoint, drive.Status.Mountpoint)
	}
}
