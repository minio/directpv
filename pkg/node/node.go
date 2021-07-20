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

	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/drive"
	"github.com/minio/direct-csi/pkg/metrics"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/sys/fs/xfs"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/volume"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func NewNodeServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*NodeServer, error) {

	kubeConfig := utils.GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return &NodeServer{}, err
		}
	}
	config.QPS = float32(utils.MaxThreadCount / 2)
	config.Burst = utils.MaxThreadCount

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
	}

	// Start background tasks
	go drive.StartController(ctx, nodeID)
	go volume.StartController(ctx, nodeID)
	go metrics.ServeMetrics(ctx, nodeID)

	return nodeServer, nil
}

type NodeServer struct {
	NodeID          string
	Identity        string
	Rack            string
	Zone            string
	Region          string
	directcsiClient clientset.Interface
	mounter         sys.VolumeMounter
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

	directCSIClient := ns.directcsiClient.DirectV1beta2()
	vclient := directCSIClient.DirectCSIVolumes()
	dclient := directCSIClient.DirectCSIDrives()
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
	xfsQuota.BlockFile = sys.GetDirectCSIPath(drive.Status.FilesystemUUID)

	quotaInfo, err := xfsQuota.GetQuota()
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Error while getting xfs volume stats: %v", err)
	}

	volUsage := &csi.VolumeUsage{
		Available: vol.Status.TotalCapacity - int64(quotaInfo.DqbCurSpace),
		Total:     vol.Status.TotalCapacity,
		Used:      int64(quotaInfo.DqbCurSpace),
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
