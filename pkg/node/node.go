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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/util/retry"
)

func NewNodeServer(ctx context.Context, identity, nodeID, rack, zone, region string, basePaths []string) (*NodeServer, error) {
	drives, err := FindDrives(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	directCSIClient := utils.GetDirectCSIClient()

	topologies := map[string]string{}
	topologies[topology.TopologyDriverIdentity] = identity
	topologies[topology.TopologyDriverNode] = nodeID
	topologies[topology.TopologyDriverRack] = rack
	topologies[topology.TopologyDriverZone] = zone
	topologies[topology.TopologyDriverRegion] = region
	driveTopology := topologies

	for _, drive := range drives {
		drive.Topology = driveTopology
		_, err := directCSIClient.DirectCSIDrives().Create(ctx, drive, metav1.CreateOptions{})
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				return nil, err
			}
		}
	}

	go startController(ctx)

	return &NodeServer{
		NodeID:    nodeID,
		Identity:  identity,
		Rack:      rack,
		Zone:      zone,
		Region:    region,
		BasePaths: basePaths,
	}, nil
}

type NodeServer struct {
	NodeID    string
	Identity  string
	Rack      string
	Zone      string
	Region    string
	BasePaths []string
}

func (n *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			topology.TopologyDriverIdentity: n.Identity,
			topology.TopologyDriverNode:     n.NodeID,
			topology.TopologyDriverRack:     n.Rack,
			topology.TopologyDriverZone:     n.Zone,
			topology.TopologyDriverRegion:   n.Region,
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             n.NodeID,
		MaxVolumesPerNode:  int64(100 * len(n.BasePaths)),
		AccessibleTopology: topology,
	}, nil
}

func (n *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nodeCap := func(cap csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
		glog.Infof("Using node capability %v", cap)

		return &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: cap,
				},
			},
		}
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			nodeCap(csi.NodeServiceCapability_RPC_GET_VOLUME_STATS),
			nodeCap(csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME),
		},
	}, nil
}

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

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dvol, cErr := directCSIClient.DirectCSIVolumes().Get(ctx, vID, metav1.GetOptions{})
		if cErr != nil {
			return status.Error(codes.NotFound, cErr.Error())
		}
		if dvol.HostPath == "" || dvol.StagingPath == "" {
			return status.Error(codes.FailedPrecondition, "volume not yet staged")
		}

		if dvol.StagingPath != stagingTargetPath {
			return status.Errorf(codes.InvalidArgument, "staging target path does not match staging path")
		}

		if dvol.ContainerPath == containerPath {
			return nil
		}
		if err := PublishVolume(ctx, stagingTargetPath, containerPath, readOnly); err != nil {
			return status.Errorf(codes.Internal, "Publish volume failed: %v", err)
		}

		copiedVolume := dvol.DeepCopy()
		copiedVolume.ContainerPath = containerPath
		if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, copiedVolume, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	vID := req.GetVolumeId()
	containerPath := req.GetTargetPath()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	if err := UnpublishVolume(ctx, containerPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	stagingTargetPath := req.GetStagingTargetPath()
	if stagingTargetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "stagingTargetPath missing in request")
	}

	directCSIClient := utils.GetDirectCSIClient()

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dvol, cErr := directCSIClient.DirectCSIVolumes().Get(ctx, vID, metav1.GetOptions{})
		if cErr != nil {
			return status.Error(codes.NotFound, cErr.Error())
		}

		if dvol.StagingPath == stagingTargetPath {
			return nil
		}

		csiDrive, err := directCSIClient.DirectCSIDrives().Get(ctx, dvol.OwnerDrive, metav1.GetOptions{})
		if err != nil {
			return status.Error(codes.NotFound, err.Error())
		}

		if csiDrive.Filesystem == "" {
			return status.Error(codes.FailedPrecondition, "Unformatted CSI Drive found")
		}

		hostPath, err := StageVolume(ctx, csiDrive, stagingTargetPath, vID)
		if err != nil {
			return status.Errorf(codes.Internal, "Staging volume failed: %v", err)
		}

		copiedVolume := dvol.DeepCopy()
		copiedVolume.HostPath = hostPath
		copiedVolume.StagingPath = stagingTargetPath
		if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, copiedVolume, metav1.UpdateOptions{}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	vID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	if err := UnstageVolume(ctx, stagingTargetPath); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (ns *NodeServer) NodeExpandVolume(ctx context.Context, in *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
