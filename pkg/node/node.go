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

	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/drive"
	"github.com/minio/direct-csi/pkg/metrics"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/volume"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NewNodeServer creates node server.
func NewNodeServer(ctx context.Context, identity, nodeID, rack, zone, region string, enableDynamicDiscovery bool) (*NodeServer, error) {
	config, err := client.GetKubeConfig()
	if err != nil {
		return &NodeServer{}, err
	}

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return &NodeServer{}, err
	}

	nodeServer := &NodeServer{
		NodeID:          nodeID,
		Identity:        identity,
		Rack:            rack,
		Zone:            zone,
		Region:          region,
		directcsiClient: directClientset,
		mounter:         &sys.DefaultVolumeMounter{},
		quotaFuncs:      &xfsQuotaFuncs{},
	}

	// Start background tasks
	go func() {
		if err := drive.StartController(ctx, nodeID); err != nil {
			klog.Error(err)
		}
	}()

	go func() {
		if err := volume.StartController(ctx, nodeID); err != nil {
			klog.Error(err)
		}
	}()

	go metrics.ServeMetrics(ctx, nodeID)
	if enableDynamicDiscovery {
		go startUeventHandler(
			ctx, nodeID, map[string]string{
				string(client.TopologyDriverIdentity): identity,
				string(client.TopologyDriverRack):     rack,
				string(client.TopologyDriverZone):     zone,
				string(client.TopologyDriverRegion):   region,
				string(client.TopologyDriverNode):     nodeID,
			},
		)
	}

	return nodeServer, nil
}

// NodeServer denotes node server.
type NodeServer struct { //revive:disable-line:exported
	NodeID          string
	Identity        string
	Rack            string
	Zone            string
	Region          string
	directcsiClient clientset.Interface
	mounter         sys.VolumeMounter
	quotaFuncs      quotaFuncs
}

//revive:enable-line:exported

// NodeGetInfo gets node information.
func (ns *NodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			string(client.TopologyDriverIdentity): ns.Identity,
			string(client.TopologyDriverRack):     ns.Rack,
			string(client.TopologyDriverZone):     ns.Zone,
			string(client.TopologyDriverRegion):   ns.Region,
			string(client.TopologyDriverNode):     ns.NodeID,
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             ns.NodeID,
		MaxVolumesPerNode:  int64(100),
		AccessibleTopology: topology,
	}, nil
}

// NodeGetCapabilities gets node capabilities.
func (ns *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nodeCap := func(cap csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
		klog.V(2).Infof("Using node capability %v", cap)

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

// NodeGetVolumeStats gets node volume stats.
func (ns *NodeServer) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	vID := req.GetVolumeId()
	volumePath := req.GetVolumePath()

	if volumePath == "" {
		return &csi.NodeGetVolumeStatsResponse{}, nil
	}

	directCSIClient := ns.directcsiClient.DirectV1beta3()
	vclient := directCSIClient.DirectCSIVolumes()
	dclient := directCSIClient.DirectCSIDrives()
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
	quota, err := ns.quotaFuncs.GetQuota(ctx, sys.GetDirectCSIPath(drive.Status.FilesystemUUID), vID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Error while getting xfs volume stats: %v", err)
	}

	volUsage := &csi.VolumeUsage{
		Available: vol.Status.TotalCapacity - int64(quota.CurrentSpace),
		Total:     vol.Status.TotalCapacity,
		Used:      int64(quota.CurrentSpace),
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

// NodeExpandVolume returns unimplemented error.
func (ns *NodeServer) NodeExpandVolume(ctx context.Context, in *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
