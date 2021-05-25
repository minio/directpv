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
	"strings"

	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/drive"
	"github.com/minio/direct-csi/pkg/metrics"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/sys/xfs"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/volume"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewNodeServer(ctx context.Context, identity, nodeID, rack, zone, region string, basePaths []string, procfs string, loopBackOnly bool) (*NodeServer, error) {

	drives, err := findDrives(ctx, nodeID, procfs, loopBackOnly)
	if err != nil {
		return nil, err
	}

	kubeConfig := utils.GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return &NodeServer{}, err
		}
	}

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return &NodeServer{}, err
	}
	directCSIClient := directClientset.DirectV1beta1()

	topologies := map[string]string{}
	topologies[topology.TopologyDriverIdentity] = identity
	topologies[topology.TopologyDriverRack] = rack
	topologies[topology.TopologyDriverZone] = zone
	topologies[topology.TopologyDriverRegion] = region
	topologies[topology.TopologyDriverNode] = nodeID
	driveTopology := topologies

	mounts, err := sys.ProbeMountInfo()
	if err != nil {
		return nil, err
	}

	driveMounter := &sys.DefaultDriveMounter{}

	for _, drive := range drives {
		isMounted := false
		for _, mount := range mounts {
			if drive.Status.Path == mount.MountSource {
				isMounted = true
				break
			}
		}

		if !isMounted {
			if drive.Status.Filesystem != "" && strings.HasPrefix(drive.Status.Mountpoint, sys.MountRoot) {
				if err := driveMounter.MountDrive(drive.Status.Path, drive.Status.Mountpoint, drive.Status.MountOptions); err != nil {
					return nil, err
				}
			}
		}

		drive.Status.Topology = driveTopology
		drive.Status.AccessTier = directcsi.AccessTierUnknown
		driveClient := directCSIClient.DirectCSIDrives()

		driveUpdate := func() error {
			existingDrive, err := driveClient.Get(ctx, drive.Name, metav1.GetOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
			})
			if err != nil {
				return err
			}
			updateOpts := metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
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
	go drive.StartDriveController(ctx, nodeID)
	go volume.StartVolumeController(ctx, nodeID)
	// Check if the volume objects are migrated and CRDs versions are in-sync
	go volume.SyncVolumeCRDVersions(ctx, nodeID)
	go metrics.ServeMetrics(ctx, nodeID)

	return &NodeServer{
		NodeID:          nodeID,
		Identity:        identity,
		Rack:            rack,
		Zone:            zone,
		Region:          region,
		directcsiClient: directClientset,
		mounter:         &sys.DefaultVolumeMounter{},
	}, nil
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
