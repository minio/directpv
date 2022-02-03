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
	"path/filepath"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/fs/xfs"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NodeStageVolume is node stage volume request handler.
func (n *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
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
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	drive, err := dclient.Get(ctx, vol.Status.Drive, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err := n.checkDrive(ctx, drive, req.GetVolumeId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	path := filepath.Join(drive.Status.Mountpoint, vID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	if err := n.safeBindMount(path, stagingTargetPath, false, false); err != nil {
		return nil, status.Errorf(codes.Internal, "failed stage volume: %v", err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(vol.Status.TotalCapacity),
		SoftLimit: uint64(vol.Status.TotalCapacity),
	}

	device, err := n.getDevice(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Unable to find device for major/minor %v:%v; %v", drive.Status.MajorNumber, drive.Status.MinorNumber, err)
	}
	if err := n.setQuota(ctx, device, stagingTargetPath, vID, quota); err != nil {
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
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
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
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.NodeUnstageVolumeResponse{}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := n.safeUnmount(stagingTargetPath, true, true, false); err != nil {
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
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	}); err != nil {
		return nil, err
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}
