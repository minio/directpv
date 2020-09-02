// This file is part of MinIO Kubernetes Cloud
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
	"github.com/minio/direct-csi/pkg/volume"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const MaxVolumes = 10000

func NewNodeServer(identity, nodeID, rack, zone, region string, basePaths []string) (*NodeServer, error) {
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
	ro := req.GetReadonly()
	targetPath := req.GetTargetPath()
	stagingPath := req.GetStagingTargetPath()
	vCtx := req.GetVolumeContext()
	vCap := req.GetVolumeCapability()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	vol, err := volume.GetVolume(ctx, vID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if vol.StagingPath != stagingPath {
		return nil, status.Error(codes.FailedPrecondition, "volume staging target path is empty or incorrect")
	}

	if access, ok := vol.ContainsTargetPaths(targetPath); ok {
		if access.Matches(req) {
			return &csi.NodePublishVolumeResponse{}, nil
		}
		return nil, status.Error(codes.AlreadyExists, "cannot reprovision volume at same path but different parameters")
	}

	if vCap == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}

	if vCap.GetBlock() != nil && vCap.GetMount() != nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability request contains both mount and block access")
	}

	if vCap.GetBlock() == nil && vCap.GetMount() == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability request contains neither mount and block access")
	}

	if vCap.GetBlock() != nil {
		if !vol.IsBlockAccessible() {
			return nil, status.Error(codes.InvalidArgument, "volume does not support block access")
		}

		if err := vol.Bind(ctx, targetPath, ro, vCtx); err != nil {
			if _, ok := status.FromError(err); ok {
				return nil, err
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		glog.V(5).Infof("published block access request for volume %s successfully", vID)
	}

	if vMount := vCap.GetMount(); vMount != nil {
		if !vol.IsMountAccessible() {
			return nil, status.Error(codes.InvalidArgument, "volume does not support mount access")
		}

		fs := vMount.GetFsType()
		flags := vMount.GetMountFlags()

		if err := vol.Mount(ctx, targetPath, fs, flags, ro, vCtx); err != nil {
			if _, ok := status.FromError(err); ok {
				return nil, err
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		glog.V(5).Infof("published mount access request for volume %s successfully", vID)
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	vID := req.GetVolumeId()
	targetPath := req.GetTargetPath()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	vol, err := volume.GetVolume(ctx, vID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err := vol.UnpublishVolume(ctx, targetPath); err != nil {
		if _, ok := status.FromError(err); ok {
			return nil, err
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (n *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	vID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	vol, err := volume.GetVolume(ctx, vID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	vol.NodeID = n.NodeID

	err = vol.StageVolume(ctx, vID, stagingTargetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Stage Volume Failed: %v", err)
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (n *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	vID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	vol, err := volume.GetVolume(ctx, vID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err := vol.UnstageVolume(ctx, vID, stagingTargetPath); err != nil {
		return nil, status.Errorf(codes.Internal, "Unstage Volume failed: %v", err)
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (ns *NodeServer) NodeExpandVolume(ctx context.Context, in *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
