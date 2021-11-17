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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/fs/xfs"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/direct-csi/pkg/sys"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func (n *NodeServer) nodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest, probeMounts func() (map[string][]sys.MountInfo, error)) (*csi.NodeStageVolumeResponse, error) {
	klog.V(3).InfoS("NodeStageVolumeRequest",
		"volumeID", req.GetVolumeId(),
		"StagingTargetPath", req.GetStagingTargetPath())

	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	directCSIClient := n.directcsiClient.DirectV1beta3()
	dclient := directCSIClient.DirectCSIDrives()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	drive, err := dclient.Get(ctx, vol.Status.Drive, metav1.GetOptions{
		TypeMeta: client.DirectCSIDriveTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err := checkDrive(drive, req.GetVolumeId(), probeMounts); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	path := filepath.Join(drive.Status.Mountpoint, vID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	if err := n.mounter.MountVolume(ctx, path, stagingTargetPath, false); err != nil {
		return nil, status.Errorf(codes.Internal, "failed stage volume: %v", err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(vol.Status.TotalCapacity),
		SoftLimit: uint64(vol.Status.TotalCapacity),
	}
	if err := n.quotaFuncs.SetQuota(ctx, sys.GetDirectCSIPath(drive.Status.FilesystemUUID), stagingTargetPath, vID, quota); err != nil {
		return nil, status.Errorf(codes.Internal, "Error while setting xfs limits: %v", err)
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directcsi.DirectCSIVolumeConditionReady):
			conditions[i].Status = utils.BoolToCondition(true)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonReady)
		case string(directcsi.DirectCSIVolumeConditionPublished):
		case string(directcsi.DirectCSIVolumeConditionStaged):
			conditions[i].Status = utils.BoolToCondition(true)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonInUse)
		}
	}

	vol.Status.HostPath = path
	vol.Status.StagingPath = stagingTargetPath

	if _, err := vclient.Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

// NodeStageVolume is node stage volume request handler.
func (n *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return n.nodeStageVolume(ctx, req, sys.ProbeMounts)
}

// NodeUnstageVolume is node unstage volume request handler.
func (n *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	klog.V(3).InfoS("NodeUnstageVolumeRequest",
		"volumeID", req.GetVolumeId(),
		"StagingTargetPath", req.GetStagingTargetPath())
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	directCSIClient := n.directcsiClient.DirectV1beta3()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.NodeUnstageVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := n.mounter.UnmountVolume(stagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directcsi.DirectCSIVolumeConditionPublished):
		case string(directcsi.DirectCSIVolumeConditionStaged):
			conditions[i].Status = utils.BoolToCondition(false)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonNotInUse)
		case string(directcsi.DirectCSIVolumeConditionReady):
			conditions[i].Status = utils.BoolToCondition(false)
			conditions[i].Reason = string(directcsi.DirectCSIVolumeReasonNotReady)
		}
	}

	vol.Status.StagingPath = ""
	if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}
