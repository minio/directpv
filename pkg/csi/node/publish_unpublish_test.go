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
	"os"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodePublishVolume(t *testing.T) {
	req := &csi.NodePublishVolumeRequest{
		VolumeId:          "volume-id-1",
		StagingTargetPath: "volume-id-1-staging-target-path",
		TargetPath:        "volume-id-1-target-path",
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "xfs"}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	nodeID := directpvtypes.NodeID("node-1")
	driveID := directpvtypes.DriveID("drive-1")
	driveName := directpvtypes.DriveName("sda")
	accessTier := directpvtypes.AccessTierDefault
	volume := types.NewVolume("volume-id-1", "fsuuid-1", nodeID, driveID, driveName, 100)
	volume.Status = types.VolumeStatus{StagingTargetPath: "volume-id-1-staging-target-path"}
	drive := types.NewDrive(driveID, types.DriveStatus{}, nodeID, driveName, accessTier)

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(drive, volume))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	nodeServer := createFakeServer()
	nodeServer.getMounts = func() (*sys.MountInfo, error) { return sys.FakeMountInfo(), nil }
	if _, err := nodeServer.NodePublishVolume(t.Context(), req); err == nil {
		t.Fatalf("expected error, but succeeded")
	}
}

func TestPublishUnpublishVolume(t *testing.T) {
	testStagingPath := t.TempDir()
	defer os.RemoveAll(testStagingPath)

	testTargetPath := t.TempDir()
	defer os.RemoveAll(testTargetPath)

	nodeID := directpvtypes.NodeID("node-1")
	driveID := directpvtypes.DriveID("drive-1")
	driveName := directpvtypes.DriveName("sda")
	accessTier := directpvtypes.AccessTierDefault
	volume := types.NewVolume("test_volume_50MB", "fsuuid-1", nodeID, driveID, driveName, 50*MiB)
	volume.Status.StagingTargetPath = testStagingPath
	drive := types.NewDrive(driveID, types.DriveStatus{}, nodeID, driveName, accessTier)

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(drive, volume))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())

	publishVolumeRequest := csi.NodePublishVolumeRequest{
		VolumeId:          volume.Name,
		StagingTargetPath: testStagingPath,
		TargetPath:        testTargetPath,
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{
				Mount: &csi.VolumeCapability_MountVolume{
					FsType: "xfs",
				},
			},
			AccessMode: &csi.VolumeCapability_AccessMode{
				Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
			},
		},
		Readonly: false,
	}

	unpublishVolumeRequest := csi.NodeUnpublishVolumeRequest{
		VolumeId:   volume.Name,
		TargetPath: testTargetPath,
	}

	ctx := t.Context()
	ns := createFakeServer()

	// Publish volume test
	ns.getMounts = func() (mountInfo *sys.MountInfo, err error) {
		mountInfo = sys.FakeMountInfo(sys.MountEntry{MountPoint: testStagingPath})
		return
	}
	_, err := ns.NodePublishVolume(ctx, &publishVolumeRequest)
	if err != nil {
		t.Fatalf("[%s] PublishVolume failed. Error: %v", publishVolumeRequest.VolumeId, err)
	}

	volObj, gErr := client.VolumeClient().Get(ctx, publishVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", publishVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.TargetPath != testTargetPath {
		t.Errorf("Wrong target path set in the volume object. Expected %v, Got: %v", testTargetPath, volObj.Status.TargetPath)
	}

	// Unpublish volume test
	if _, err := ns.NodeUnpublishVolume(ctx, &unpublishVolumeRequest); err != nil {
		t.Fatalf("[%s] PublishVolume failed. Error: %v", unpublishVolumeRequest.VolumeId, err)
	}

	volObj, gErr = client.VolumeClient().Get(ctx, unpublishVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", unpublishVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if the status fields were unset
	if volObj.Status.TargetPath != "" {
		t.Errorf("StagingPath was not set to empty. Got: %v", volObj.Status.TargetPath)
	}
}
