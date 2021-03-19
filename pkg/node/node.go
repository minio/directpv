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

	"github.com/minio/direct-csi/pkg/drive"
	"github.com/minio/direct-csi/pkg/sys/xfs"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/volume"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewNodeServer(ctx context.Context, identity, nodeID, rack, zone, region string, basePaths []string, procfs string, crdVersion string) (*NodeServer, error) {

	drives, err := findDrives(ctx, nodeID, procfs)
	if err != nil {
		return nil, err
	}
	directCSIClient := utils.GetDirectCSIClient()

	topologies := map[string]string{}
	topologies[topology.TopologyDriverIdentity] = identity
	topologies[topology.TopologyDriverRack] = rack
	topologies[topology.TopologyDriverZone] = zone
	topologies[topology.TopologyDriverRegion] = region
	topologies[topology.TopologyDriverNode] = nodeID
	driveTopology := topologies

	for _, drive := range drives {
		drive.Status.Topology = driveTopology
		drive.Status.AccessTier = directcsi.AccessTierUnknown
		driveClient := directCSIClient.DirectCSIDrives()

		driveUpdate := func() error {
			existingDrive, err := driveClient.Get(ctx, drive.Name, metav1.GetOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(crdVersion),
			})
			if err != nil {
				return err
			}
			updateOpts := metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(crdVersion),
			}
			_, err = driveClient.Update(ctx, existingDrive, updateOpts)
			return err
		}

		// Do a dummy update to ensure the drive object is migrated to the latest CRD version
		if err := retry.RetryOnConflict(retry.DefaultRetry, driveUpdate); err != nil {
			if !errors.IsNotFound(err) {
				return nil, err
			}
			if _, err := driveClient.Create(ctx, &drive, metav1.CreateOptions{}); err != nil {
				return nil, err
			}
		}
	}

	// Start background tasks
	go drive.StartDriveController(ctx, nodeID, crdVersion)
	go volume.StartVolumeController(ctx, nodeID, crdVersion)
	// Check if the volume objects are migrated and CRDs versions are in-sync
	go volume.SyncVolumeCRDVersions(ctx, nodeID, crdVersion)

	return &NodeServer{
		NodeID:     nodeID,
		Identity:   identity,
		Rack:       rack,
		Zone:       zone,
		Region:     region,
		CRDVersion: crdVersion,
	}, nil
}

type NodeServer struct {
	NodeID     string
	Identity   string
	Rack       string
	Zone       string
	Region     string
	CRDVersion string
}

func (n *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			topology.TopologyDriverIdentity: n.Identity,
			topology.TopologyDriverRack:     n.Rack,
			topology.TopologyDriverZone:     n.Zone,
			topology.TopologyDriverRegion:   n.Region,
			topology.TopologyDriverNode:     n.NodeID,
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             n.NodeID,
		MaxVolumesPerNode:  int64(100),
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

func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	vID := req.GetVolumeId()
	volumePath := req.GetVolumePath()

	if volumePath == "" {
		return &csi.NodeGetVolumeStatsResponse{}, nil
	}

	xfsQuota := &xfs.XFSQuota{
		Path:      volumePath,
		ProjectID: vID,
	}
	volStats, err := xfsQuota.GetVolumeStats(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error while getting xfs volume stats: %v", err)
	}

	volUsage := &csi.VolumeUsage{
		Available: volStats.AvailableBytes,
		Total:     volStats.TotalBytes,
		Used:      volStats.UsedBytes,
		Unit:      csi.VolumeUsage_BYTES,
	}

	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			volUsage,
		},
		VolumeCondition: &csi.VolumeCondition{
			Abnormal: false,
			Message:  "",
		},
	}, nil
}

func (ns *NodeServer) NodeExpandVolume(ctx context.Context, in *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
