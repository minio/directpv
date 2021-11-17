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

package node

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KB = 1 << 10
	MB = KB << 10

	mb50  = 50 * MB
	mb100 = 100 * MB
	mb20  = 20 * MB
)

func TestNodeStageVolume(t *testing.T) {
	case1Req := &csi.NodeStageVolumeRequest{
		VolumeId:          "volume-id-1",
		StagingTargetPath: "volume-id-1-staging-target-path",
		VolumeCapability: &csi.VolumeCapability{
			AccessType: &csi.VolumeCapability_Mount{Mount: &csi.VolumeCapability_MountVolume{FsType: "xfs"}},
			AccessMode: &csi.VolumeCapability_AccessMode{Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER},
		},
	}

	case1Drive := &directcsi.DirectCSIDrive{
		TypeMeta:   client.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
	}

	case2Drive := &directcsi.DirectCSIDrive{
		TypeMeta:   client.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
		Status:     directcsi.DirectCSIDriveStatus{DriveStatus: directcsi.DriveStatusInUse},
	}

	case3Drive := &directcsi.DirectCSIDrive{
		TypeMeta: client.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:       "drive-1",
			Finalizers: []string{directcsi.DirectCSIDriveFinalizerPrefix + "volume-id-1"},
		},
		Status: directcsi.DirectCSIDriveStatus{DriveStatus: directcsi.DriveStatusInUse},
	}

	testCases := []struct {
		req       *csi.NodeStageVolumeRequest
		drive     *directcsi.DirectCSIDrive
		mountInfo map[string][]sys.MountInfo
	}{
		{case1Req, case1Drive, nil},
		{case1Req, case2Drive, nil},
		{case1Req, case3Drive, map[string][]sys.MountInfo{"1:0": {}}},
		{case1Req, case3Drive, map[string][]sys.MountInfo{"0:0": {}}},
	}

	for i, testCase := range testCases {
		volume := &directcsi.DirectCSIVolume{
			TypeMeta:   client.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{Name: testCase.req.VolumeId},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      testNodeName,
				Drive:         testCase.drive.Name,
				TotalCapacity: 100 * MB,
			},
		}

		nodeServer := createFakeNodeServer()
		nodeServer.directcsiClient = fakedirect.NewSimpleClientset(volume, testCase.drive)
		_, err := nodeServer.nodeStageVolume(
			context.TODO(),
			testCase.req,
			func() (map[string][]sys.MountInfo, error) { return testCase.mountInfo, nil },
		)
		if err == nil {
			t.Fatalf("case %v: expected error, but succeeded", i+1)
		}
	}
}

func TestStageUnstageVolume(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName50MB := "test_volume_50MB"

	testMountPointDir, err := os.MkdirTemp("", "test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testMountPointDir)

	testObjects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			TypeMeta: client.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testDriveName,
				Finalizers: []string{
					string(directcsi.DirectCSIDriveFinalizerDataProtection),
					directcsi.DirectCSIDriveFinalizerPrefix + testVolumeName50MB,
				},
			},
			Status: directcsi.DirectCSIDriveStatus{
				Mountpoint:        testMountPointDir,
				NodeName:          testNodeName,
				DriveStatus:       directcsi.DriveStatusInUse,
				FreeCapacity:      mb50,
				AllocatedCapacity: mb50,
				TotalCapacity:     mb100,
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: client.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName50MB,
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      testNodeName,
				Drive:         testDriveName,
				TotalCapacity: mb20,
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

	stageVolumeRequest := csi.NodeStageVolumeRequest{
		VolumeId:          testVolumeName50MB,
		StagingTargetPath: "/path/to/target",
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
	}

	unstageVolumeRequest := csi.NodeUnstageVolumeRequest{
		VolumeId:          testVolumeName50MB,
		StagingTargetPath: "/path/to/target",
	}

	ctx := context.TODO()
	ns := createFakeNodeServer()
	ns.directcsiClient = fakedirect.NewSimpleClientset(testObjects...)
	directCSIClient := ns.directcsiClient.DirectV1beta3()
	hostPath := filepath.Join(testMountPointDir, testVolumeName50MB)

	// Stage Volume test
	_, err = ns.nodeStageVolume(ctx, &stageVolumeRequest, func() (map[string][]sys.MountInfo, error) {
		return map[string][]sys.MountInfo{"0:0": {{MountPoint: "/var/lib/direct-csi/mnt"}}}, nil
	})
	if err != nil {
		t.Fatalf("[%s] StageVolume failed. Error: %v", stageVolumeRequest.VolumeId, err)
	}

	volObj, gErr := directCSIClient.DirectCSIVolumes().Get(ctx, stageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", stageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if mount arguments were passed correctly
	if ns.mounter.(*fakeVolumeMounter).mountArgs.source != hostPath {
		t.Errorf("Wrong source argument passed for mounting. Expected: %v, Got: %v", filepath.Join(testMountPointDir, testVolumeName50MB), ns.mounter.(*fakeVolumeMounter).mountArgs.source)
	}
	if ns.mounter.(*fakeVolumeMounter).mountArgs.destination != stageVolumeRequest.GetStagingTargetPath() {
		t.Errorf("Wrong destination argument passed for mounting. Expected: %v, Got: %v", stageVolumeRequest.GetStagingTargetPath(), ns.mounter.(*fakeVolumeMounter).mountArgs.destination)
	}
	if ns.mounter.(*fakeVolumeMounter).mountArgs.readOnly {
		t.Errorf("Wrong readOnly argument passed for mounting. Expected: False, Got: %v", ns.mounter.(*fakeVolumeMounter).mountArgs.readOnly)
	}

	// Check if status fields were set correctly
	if volObj.Status.HostPath != hostPath {
		t.Errorf("Wrong HostPath set in the volume object. Expected %v, Got: %v", hostPath, volObj.Status.HostPath)
	}
	if volObj.Status.StagingPath != stageVolumeRequest.GetStagingTargetPath() {
		t.Errorf("Wrong StagingPath set in the volume object. Expected %v, Got: %v", stageVolumeRequest.GetStagingTargetPath(), volObj.Status.StagingPath)
	}

	// Check if conditions were toggled correctly
	if !utils.IsCondition(volObj.Status.Conditions, string(directcsi.DirectCSIVolumeConditionStaged), metav1.ConditionTrue, string(directcsi.DirectCSIVolumeReasonInUse), "") {
		t.Errorf("unexpected status.conditions after staging = %v", volObj.Status.Conditions)
	}

	// Unstage Volume test
	if _, err := ns.NodeUnstageVolume(ctx, &unstageVolumeRequest); err != nil {
		t.Fatalf("[%s] UnstageVolume failed. Error: %v", unstageVolumeRequest.VolumeId, err)
	}

	volObj, gErr = directCSIClient.DirectCSIVolumes().Get(ctx, unstageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", unstageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if unmount arguments were set correctly
	if ns.mounter.(*fakeVolumeMounter).unmountArgs.target != unstageVolumeRequest.GetStagingTargetPath() {
		t.Errorf("Wrong target argument passed for unmounting. Expected: %v, Got: %v", unstageVolumeRequest.GetStagingTargetPath(), ns.mounter.(*fakeVolumeMounter).unmountArgs.target)
	}

	// Check if status fields were set correctly
	if volObj.Status.StagingPath != "" {
		t.Errorf("StagingPath was not set to empty. Got: %v", volObj.Status.StagingPath)
	}

	// Check if conditions were toggled correctly
	if !utils.IsCondition(volObj.Status.Conditions, string(directcsi.DirectCSIVolumeConditionStaged), metav1.ConditionFalse, string(directcsi.DirectCSIVolumeReasonNotInUse), "") {
		t.Errorf("unexpected status.conditions after unstaging = %v", volObj.Status.Conditions)
	}
}
