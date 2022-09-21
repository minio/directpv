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
	"fmt"
	"path"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const MiB = 1024 * 1024

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
		Status:     types.DriveStatus{Status: directpvtypes.DriveStatusOK},
	}

	case3Drive := &types.Drive{
		TypeMeta: types.NewDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:       "drive-1",
			Finalizers: []string{consts.DriveFinalizerPrefix + "volume-id-1"},
		},
		Status: types.DriveStatus{Status: directpvtypes.DriveStatusOK},
	}

	testCases := []struct {
		req       *csi.NodeStageVolumeRequest
		drive     *types.Drive
		mountInfo map[string][]string
	}{
		{case1Req, case1Drive, nil},
		{case1Req, case2Drive, nil},
		{case1Req, case3Drive, map[string][]string{consts.MountRootDir: {}}},
		{case1Req, case3Drive, map[string][]string{consts.MountRootDir: {}}},
	}

	for i, testCase := range testCases {
		volume := &types.Volume{
			TypeMeta:   types.NewVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{Name: testCase.req.VolumeId},
			Status: types.VolumeStatus{
				NodeName:      testNodeName,
				DriveName:     testCase.drive.Name,
				TotalCapacity: 100 * MiB,
			},
		}

		clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume, testCase.drive))
		client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
		client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

		nodeServer := createFakeServer()
		nodeServer.getMounts = func() (map[string][]string, map[string][]string, error) {
			return testCase.mountInfo, nil, nil
		}
		nodeServer.bindMount = func(source, stagingTargetPath string, readOnly bool) error {
			if _, found := testCase.mountInfo[source]; !found {
				return fmt.Errorf("source is not mounted")
			}
			return nil
		}
		if _, err := nodeServer.NodeStageVolume(context.TODO(), testCase.req); err == nil {
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
				NodeName:          testNodeName,
				Status:            directpvtypes.DriveStatusOK,
				FreeCapacity:      50 * MiB,
				AllocatedCapacity: 50 * MiB,
				TotalCapacity:     100 * MiB,
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
				DriveName:     testDriveName,
				FSUUID:        testDriveName,
				TotalCapacity: 20 * MiB,
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

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testObjects...))
	client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	ctx := context.TODO()
	ns := createFakeServer()
	dataPath := path.Join(consts.MountRootDir, testDriveName, ".FSUUID."+testDriveName, testVolumeName50MB)
	ns.getMounts = func() (map[string][]string, map[string][]string, error) {
		return map[string][]string{consts.MountRootDir: {}}, nil, nil
	}

	// Stage Volume test
	_, err := ns.NodeStageVolume(ctx, &stageVolumeRequest)
	if err != nil {
		t.Fatalf("[%s] StageVolume failed. Error: %v", stageVolumeRequest.VolumeId, err)
	}

	volObj, gErr := client.VolumeClient().Get(ctx, stageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", stageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.DataPath != dataPath {
		t.Errorf("Wrong HostPath set in the volume object. Expected %v, Got: %v", dataPath, volObj.Status.DataPath)
	}
	if volObj.Status.StagingTargetPath != stageVolumeRequest.GetStagingTargetPath() {
		t.Errorf("Wrong StagingTargetPath set in the volume object. Expected %v, Got: %v", stageVolumeRequest.GetStagingTargetPath(), volObj.Status.StagingTargetPath)
	}

	// Unstage Volume test
	if _, err := ns.NodeUnstageVolume(ctx, &unstageVolumeRequest); err != nil {
		t.Fatalf("[%s] UnstageVolume failed. Error: %v", unstageVolumeRequest.VolumeId, err)
	}

	volObj, gErr = client.VolumeClient().Get(ctx, unstageVolumeRequest.GetVolumeId(), metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if gErr != nil {
		t.Fatalf("Volume (%s) not found. Error: %v", unstageVolumeRequest.GetVolumeId(), gErr)
	}

	// Check if status fields were set correctly
	if volObj.Status.StagingTargetPath != "" {
		t.Errorf("StagingTargetPath was not set to empty. Got: %v", volObj.Status.StagingTargetPath)
	}
}
