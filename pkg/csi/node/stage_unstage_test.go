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
	"errors"
	"path"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/sys"
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

	case1Drive := types.NewDrive("drive-1", types.DriveStatus{}, "node-1", "sda", directpvtypes.AccessTierDefault)
	case2Drive := types.NewDrive("drive-1", types.DriveStatus{Status: directpvtypes.DriveStatusReady}, "node-1", "sda", directpvtypes.AccessTierDefault)
	case3Drive := types.NewDrive("drive-1", types.DriveStatus{Status: directpvtypes.DriveStatusReady}, "node-1", "sda", directpvtypes.AccessTierDefault)
	case3Drive.AddVolumeFinalizer("volume-id-1")

	testCases := []struct {
		req       *csi.NodeStageVolumeRequest
		drive     *types.Drive
		mountInfo *sys.MountInfo
	}{
		{case1Req, case1Drive, sys.FakeMountInfo()},
		{case1Req, case2Drive, sys.FakeMountInfo()},
		{case1Req, case3Drive, sys.FakeMountInfo(sys.MountEntry{MountPoint: consts.MountRootDir})},
		{case1Req, case3Drive, sys.FakeMountInfo(sys.MountEntry{MountPoint: consts.MountRootDir})},
	}

	for i, testCase := range testCases {
		volume := types.NewVolume(
			testCase.req.VolumeId,
			"fsuuid1",
			testNodeName,
			"sda",
			directpvtypes.DriveName(testCase.drive.Name),
			100*MiB,
		)

		clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(volume, testCase.drive))
		client.SetDriveInterface(clientset.DirectpvLatest().DirectPVDrives())
		client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

		nodeServer := createFakeServer()
		nodeServer.getMounts = func() (*sys.MountInfo, error) {
			return testCase.mountInfo, nil
		}
		nodeServer.bindMount = func(source, _ string, _ bool) error {
			if testCase.mountInfo.FilterByMountSource(source).IsEmpty() {
				return errors.New("source is not mounted")
			}
			return nil
		}
		if _, err := nodeServer.NodeStageVolume(t.Context(), testCase.req); err == nil {
			t.Fatalf("case %v: expected error, but succeeded", i+1)
		}
	}
}

func TestStageUnstageVolume(t *testing.T) {
	testDriveName := "test_drive"
	testVolumeName50MB := "test_volume_50MB"

	drive := types.NewDrive(
		directpvtypes.DriveID(testDriveName),
		types.DriveStatus{
			TotalCapacity:     100 * MiB,
			FreeCapacity:      50 * MiB,
			AllocatedCapacity: 50 * MiB,
			Status:            directpvtypes.DriveStatusReady,
		},
		testNodeName,
		"sda",
		directpvtypes.AccessTierDefault,
	)
	drive.AddVolumeFinalizer(testVolumeName50MB)
	testObjects := []runtime.Object{
		drive,
		types.NewVolume(
			testVolumeName50MB,
			testDriveName,
			testNodeName,
			directpvtypes.DriveID(testDriveName),
			directpvtypes.DriveName(testDriveName),
			20*MiB,
		),
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

	ctx := t.Context()
	ns := createFakeServer()
	dataPath := path.Join(consts.MountRootDir, testDriveName, ".FSUUID."+testDriveName, testVolumeName50MB)
	ns.getMounts = func() (*sys.MountInfo, error) {
		return sys.FakeMountInfo(sys.MountEntry{MountSource: "/dev/", MountPoint: "/var/lib/directpv/mnt/test_drive"}), nil
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
