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
	"fmt"
	"os"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// NodeStageVolume is node stage volume request handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume
func (server *Server) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.V(3).InfoS("NodeStageVolume() called",
		"volumeID", req.GetVolumeId(),
		"StagingTargetPath", req.GetStagingTargetPath())

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	volumeDir := types.GetVolumeDir(volume.Status.FSUUID, volumeID)
	if err := server.mkdir(volumeDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		// FIXME: handle I/O error and mark associated drive's status as ERROR.
		klog.ErrorS(err, "unable to create volume directory", "VolumeDir", volumeDir)
		return nil, err
	}

	if err := server.bindMount(volumeDir, stagingTargetPath, false); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to bind mount volume directory to staging target path; %v", err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(volume.Status.TotalCapacity),
		SoftLimit: uint64(volume.Status.TotalCapacity),
	}

	device, err := server.getDeviceByFSUUID(volume.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				"on the host to reload",
			"FSUUID", volume.Status.FSUUID)
		client.Eventf(
			volume, corev1.EventTypeWarning, "NodeStageVolume",
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", volume.Status.FSUUID)
		return nil, status.Errorf(codes.Internal, "Unable to find device by FSUUID %v; %v", volume.Status.FSUUID, err)
	}

	if err := server.setQuota(ctx, device, stagingTargetPath, volumeID, quota); err != nil {
		klog.ErrorS(err, "unable to set quota on staging target path", "StagingTargetPath", stagingTargetPath)
		return nil, status.Errorf(codes.Internal, "unable to set quota on staging target path; %v", err)
	}

	volume.Status.DataPath = volumeDir
	volume.Status.StagingTargetPath = stagingTargetPath

	if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume is node unstage volume request handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunstagevolume
func (server *Server) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.V(3).InfoS("NodeUnstageVolume() called",
		"volumeID", req.GetVolumeId(),
		"StagingTargetPath", req.GetStagingTargetPath())
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &csi.NodeUnstageVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if !volume.Status.IsStaged() {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("unstage is called without stage for volume %v", volume.Name))
	}

	if volume.Status.StagingTargetPath != stagingTargetPath {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("staging target path %v doesn't match with requested staging target path %v", volume.Status.StagingTargetPath, stagingTargetPath))
	}

	if err := server.unmount(stagingTargetPath); err != nil {
		klog.ErrorS(err, "unable to unmount staging target path", "StagingTargetPath", stagingTargetPath)
		return nil, status.Error(codes.Internal, err.Error())
	}

	volume.Status.StagingTargetPath = ""
	if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}
