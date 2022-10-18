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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// NodeStageVolume is node stage volume request handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodestagevolume
func (server *Server) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	klog.V(3).InfoS("Stage volume requested",
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

	code, err := drive.StageVolume(
		ctx,
		volume,
		stagingTargetPath,
		server.getDeviceByFSUUID,
		server.mkdir,
		server.setQuota,
		server.bindMount,
	)
	if err != nil {
		return nil, status.Error(code, err.Error())
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeUnstageVolume is node unstage volume request handler.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeunstagevolume
func (server *Server) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.V(3).InfoS("Unstage volume requested",
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

	if !volume.IsStaged() {
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}
