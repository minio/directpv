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
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/utils"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
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

	volume := &directcsi.DirectCSIVolume{
		TypeMeta:   utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "volume-id-1"},
		Status:     directcsi.DirectCSIVolumeStatus{StagingPath: "volume-id-1-staging-target-path"},
	}

	nodeServer := createFakeNodeServer()
	nodeServer.directcsiClient = fakedirect.NewSimpleClientset(volume)
	_, err := nodeServer.NodePublishVolume(context.TODO(), req)
	if err == nil {
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

	testContainerPath, tErr := createTestDir("test_container_")
	if tErr != nil {
		t.Fatalf("Could not create test dirs: %v", tErr)
	}
	defer os.RemoveAll(testContainerPath)

	testVol := &directcsi.DirectCSIVolume{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: testVolumeName50MB,
			Finalizers: []string{
				string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
			},
		},
		Status: directcsi.DirectCSIVolumeStatus{
			NodeName:      testNodeName,
			StagingPath:   testStagingPath,
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
	}

	publishVolumeRequest := csi.NodePublishVolumeRequest{
		VolumeId:          testVolumeName50MB,
		StagingTargetPath: testStagingPath,
		TargetPath:        testContainerPath,
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
		TargetPath: testContainerPath,
	}

	ctx := context.TODO()
	ns := createFakeNodeServer()
	ns.directcsiClient = fakedirect.NewSimpleClientset(testVol)
	directCSIClient := ns.directcsiClient.DirectV1beta4()

	// Publish volume test
	ns.probeMounts = func() (map[string][]mount.MountInfo, error) {
		return map[string][]mount.MountInfo{"0:0": {{MountPoint: testStagingPath}}}, nil
	}
	_, err := ns.NodePublishVolume(ctx, &publishVolumeRequest)
	if err != nil {
		t.Fatalf("[%s] PublishVolume failed. Error: %v", publishVolumeRequest.VolumeId, err)
	}

	volObj, gErr := directCSIClient.DirectCSIVolumes().Get(ctx, publishVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", publishVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.ContainerPath != testContainerPath {
		t.Errorf("Wrong ContainerPath set in the volume object. Expected %v, Got: %v", testContainerPath, volObj.Status.ContainerPath)
	}

	// Check if conditions were toggled correctly
	if !utils.IsCondition(volObj.Status.Conditions, string(directcsi.DirectCSIVolumeConditionPublished), metav1.ConditionTrue, string(directcsi.DirectCSIVolumeReasonInUse), "") {
		t.Errorf("unexpected status.conditions after publishing = %v", volObj.Status.Conditions)
	}

	// Unpublish volume test
	if _, err := ns.NodeUnpublishVolume(ctx, &unpublishVolumeRequest); err != nil {
		t.Fatalf("[%s] PublishVolume failed. Error: %v", unpublishVolumeRequest.VolumeId, err)
	}

	volObj, gErr = directCSIClient.DirectCSIVolumes().Get(ctx, unpublishVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", unpublishVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if the status fields were unset
	if volObj.Status.ContainerPath != "" {
		t.Errorf("StagingPath was not set to empty. Got: %v", volObj.Status.ContainerPath)
	}

	// Check if the conditions were toggled correctly
	if !utils.IsCondition(volObj.Status.Conditions, string(directcsi.DirectCSIVolumeConditionPublished), metav1.ConditionFalse, string(directcsi.DirectCSIVolumeReasonNotInUse), "") {
		t.Errorf("unexpected status.conditions after unstaging = %v", volObj.Status.Conditions)
	}
}
