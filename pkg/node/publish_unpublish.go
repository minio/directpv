// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (n *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}
	containerPath := req.GetTargetPath()
	if containerPath == "" {
		return nil, status.Error(codes.InvalidArgument, "containerPath missing in request")
	}

	readOnly := req.GetReadonly()
	directCSIClient := utils.GetDirectCSIClient()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	// If not staged
	if vol.Status.StagingPath != stagingTargetPath {
		return nil, status.Error(codes.Internal, "cannot publish volume that hasn't been staged")
	}

	// If published
	if vol.Status.ContainerPath == containerPath {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	if err := mountVolume(ctx, stagingTargetPath, containerPath, vID, 0, readOnly); err != nil {
		return nil, status.Errorf(codes.Internal, "failed volume publish: %v", err)
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directv1alpha1.DirectCSIVolumeConditionPublished):
			conditions[i].Status = utils.BoolToCondition(true)
		case string(directv1alpha1.DirectCSIVolumeConditionStaged):
		}
	}
	vol.Status.ContainerPath = containerPath

	if _, err := vclient.Update(ctx, vol, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	containerPath := req.GetTargetPath()
	if containerPath == "" {
		return nil, status.Error(codes.InvalidArgument, "containerPath missing in request")
	}

	directCSIClient := utils.GetDirectCSIClient()
	vclient := directCSIClient.DirectCSIVolumes()
	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.NotFound, err.Error())
	}
	// If not published
	if vol.Status.ContainerPath == "" {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	if err := sys.SafeUnmount(containerPath, nil); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	conditions := vol.Status.Conditions
	for i, c := range conditions {
		switch c.Type {
		case string(directv1alpha1.DirectCSIVolumeConditionPublished):
			conditions[i].Status = utils.BoolToCondition(false)
			conditions[i].Reason = directv1alpha1.DirectCSIVolumeReasonInUse
		case string(directv1alpha1.DirectCSIVolumeConditionStaged):
		}
	}
	vol.Status.ContainerPath = ""

	if _, err := vclient.Update(ctx, vol, metav1.UpdateOptions{}); err != nil {
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}
