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
	"os"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
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

	volume := &types.Volume{
		TypeMeta:   types.NewVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "volume-id-1"},
		Status:     types.VolumeStatus{StagingTargetPath: "volume-id-1-staging-target-path"},
	}

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	nodeServer := createFakeServer()
	if _, err := nodeServer.NodePublishVolume(context.TODO(), req); err == nil {
		t.Fatalf("expected error, but succeeded")
	}
}

func TestPublishUnpublishVolume(t *testing.T) {
	testVolumeName50MB := "test_volume_50MB"

	createTestDir := func(prefix string) (string, error) {
		tDir, err := os.MkdirTemp("", prefix)
		if err != nil {
			return "", err
		}
		return tDir, nil
	}

	testStagingPath, tErr := createTestDir("test_staging_")
	if tErr != nil {
		t.Fatalf("Could not create test dirs: %v", tErr)
	}
	defer os.RemoveAll(testStagingPath)

	testTargetPath, tErr := createTestDir("test_target_")
	if tErr != nil {
		t.Fatalf("Could not create test dirs: %v", tErr)
	}
	defer os.RemoveAll(testTargetPath)

	testVol := types.NewVolume(testVolumeName50MB, "fsuuid1", testNodeName, "sda", "sda", 20*MiB)
	testVol.Status.StagingTargetPath = testStagingPath

	publishVolumeRequest := csi.NodePublishVolumeRequest{
		VolumeId:          testVolumeName50MB,
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
		VolumeId:   testVolumeName50MB,
		TargetPath: testTargetPath,
	}

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testVol))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	ctx := context.TODO()
	ns := createFakeServer()

	// Publish volume test
	ns.getMounts = func() (map[string]utils.StringSet, error) {
		return map[string]utils.StringSet{testStagingPath: nil}, nil
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
