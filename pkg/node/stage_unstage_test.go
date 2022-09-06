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
	"errors"
	"path"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	case1Drive := &types.Drive{
		TypeMeta:   types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
	}

	case2Drive := &types.Drive{
		TypeMeta:   types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{Name: "drive-1"},
		Status:     types.DriveStatus{DriveStatus: directcsi.DriveStatusInUse},
	}

	case3Drive := &types.Drive{
		TypeMeta: types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:       "drive-1",
			Finalizers: []string{consts.DriveFinalizerPrefix + "volume-id-1"},
		},
		Status: types.DriveStatus{DriveStatus: directcsi.DriveStatusInUse},
	}

	testCases := []struct {
		req       *csi.NodeStageVolumeRequest
		drive     *types.Drive
		mountInfo map[string][]mount.MountInfo
	}{
		{case1Req, case1Drive, nil},
		{case1Req, case2Drive, nil},
		{case1Req, case3Drive, map[string][]mount.MountInfo{"1:0": {}}},
		{case1Req, case3Drive, map[string][]mount.MountInfo{"0:0": {}}},
	}

	for i, testCase := range testCases {
		volume := &types.Volume{
			TypeMeta:   types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{Name: testCase.req.VolumeId},
			Status: types.VolumeStatus{
				NodeName:      testNodeName,
				Drive:         testCase.drive.Name,
				TotalCapacity: 100 * MB,
			},
		}

		nodeServer := createFakeServer()
		nodeServer.directcsiClient = fakedirect.NewSimpleClientset(volume, testCase.drive)
		nodeServer.probeMounts = func() (map[string][]mount.MountInfo, error) {
			return testCase.mountInfo, nil
		}
		nodeServer.verifyHostStateForDrive = func(drive *types.Drive) error {
			if drive.Status.Path == "" {
				return errors.New("empty path")
			}
			return nil
		}
		_, err := nodeServer.NodeStageVolume(context.TODO(), testCase.req)
		if err == nil {
			t.Fatalf("case %v: expected error, but succeeded", i+1)
		}
	}
}

func TestStageUnstageVolume(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName50MB := "test_volume_50MB"

	testObjects := []runtime.Object{
		&types.Drive{
			TypeMeta: types.NewDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testDriveName,
				Finalizers: []string{
					string(consts.DriveFinalizerDataProtection),
					consts.DriveFinalizerPrefix + testVolumeName50MB,
				},
			},
			Status: types.DriveStatus{
				Mountpoint:        path.Join(sys.MountRoot, testDriveName),
				NodeName:          testNodeName,
				DriveStatus:       directcsi.DriveStatusInUse,
				FreeCapacity:      mb50,
				AllocatedCapacity: mb50,
				TotalCapacity:     mb100,
			},
		},
		&types.Volume{
			TypeMeta: types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: testVolumeName50MB,
				Finalizers: []string{
					string(consts.VolumeFinalizerPurgeProtection),
				},
			},
			Status: types.VolumeStatus{
				NodeName:      testNodeName,
				Drive:         testDriveName,
				TotalCapacity: mb20,
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
						Reason:             string(directpvtypes.VolumeConditionReasonReady),
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
	ns := createFakeServer()
	ns.directcsiClient = fakedirect.NewSimpleClientset(testObjects...)
	directCSIClient := ns.directcsiClient.DirectV1beta4()
	hostPath := path.Join(
		path.Join(
			sys.MountRoot,
			testDriveName,
		),
		testVolumeName50MB)
	ns.probeMounts = func() (map[string][]mount.MountInfo, error) {
		return map[string][]mount.MountInfo{"0:0": {{MountPoint: "/var/lib/direct-csi/mnt", MajorMinor: "0:0"}}}, nil
	}

	// Stage Volume test
	_, err := ns.NodeStageVolume(ctx, &stageVolumeRequest)
	if err != nil {
		t.Fatalf("[%s] StageVolume failed. Error: %v", stageVolumeRequest.VolumeId, err)
	}

	volObj, gErr := directCSIClient.DirectCSIVolumes().Get(ctx, stageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", stageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.HostPath != hostPath {
		t.Errorf("Wrong HostPath set in the volume object. Expected %v, Got: %v", hostPath, volObj.Status.HostPath)
	}
	if volObj.Status.StagingPath != stageVolumeRequest.GetStagingTargetPath() {
		t.Errorf("Wrong StagingPath set in the volume object. Expected %v, Got: %v", stageVolumeRequest.GetStagingTargetPath(), volObj.Status.StagingPath)
	}

	// Check if conditions were toggled correctly
	if !k8s.IsCondition(volObj.Status.Conditions, string(directpvtypes.VolumeConditionTypeStaged), metav1.ConditionTrue, string(directpvtypes.VolumeConditionReasonInUse), "") {
		t.Errorf("unexpected status.conditions after staging = %v", volObj.Status.Conditions)
	}

	// Unstage Volume test
	if _, err := ns.NodeUnstageVolume(ctx, &unstageVolumeRequest); err != nil {
		t.Fatalf("[%s] UnstageVolume failed. Error: %v", unstageVolumeRequest.VolumeId, err)
	}

	volObj, gErr = directCSIClient.DirectCSIVolumes().Get(ctx, unstageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", unstageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.StagingPath != "" {
		t.Errorf("StagingPath was not set to empty. Got: %v", volObj.Status.StagingPath)
	}

	// Check if conditions were toggled correctly
	if !k8s.IsCondition(volObj.Status.Conditions, string(directpvtypes.VolumeConditionTypeStaged), metav1.ConditionFalse, string(directpvtypes.VolumeConditionReasonNotInUse), "") {
		t.Errorf("unexpected status.conditions after unstaging = %v", volObj.Status.Conditions)
	}
}
