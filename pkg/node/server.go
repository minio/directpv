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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/metrics"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// Server denotes node server.
type Server struct {
	nodeID   string
	identity string
	rack     string
	zone     string
	region   string

	getMounts         func() (map[string][]string, map[string][]string, error)
	getDeviceByFSUUID func(fsuuid string) (string, error)
	bindMount         func(source, target string, readOnly bool) error
	unmount           func(target string) error
	getQuota          func(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error)
	setQuota          func(ctx context.Context, device, path, volumeID string, quota xfs.Quota) (err error)
	mkdir             func(path string, perm os.FileMode) error
}

// NewServer creates node server.
func NewServer(ctx context.Context,
	identity, nodeID, rack, zone, region string,
	metricsPort int,
) (*Server, error) {
	nodeServer := &Server{
		nodeID:   nodeID,
		identity: identity,
		rack:     rack,
		zone:     zone,
		region:   region,

		getMounts:         sys.GetMounts,
		getDeviceByFSUUID: device.GetDeviceByFSUUID,
		bindMount:         xfs.BindMount,
		unmount:           func(target string) error { return sys.SafeUnmount(target, true, true, false) },
		getQuota:          xfs.GetQuota,
		setQuota:          xfs.SetQuota,
		mkdir:             os.Mkdir,
	}

	go metrics.ServeMetrics(ctx, nodeID, metricsPort)

	return nodeServer, nil
}

// NodeGetInfo gets node information.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetinfo
func (server *Server) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			string(types.TopologyDriverIdentity): server.identity,
			string(types.TopologyDriverRack):     server.rack,
			string(types.TopologyDriverZone):     server.zone,
			string(types.TopologyDriverRegion):   server.region,
			string(types.TopologyDriverNode):     server.nodeID,
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             server.nodeID,
		MaxVolumesPerNode:  100,
		AccessibleTopology: topology,
	}, nil
}

// NodeGetCapabilities gets node capabilities.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetcapabilities
func (server *Server) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	nodeCap := func(cap csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
		klog.V(5).InfoS("Using node capability", "NodeServiceCapability", cap)

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
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetvolumestats
func (server *Server) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	volumeID := req.GetVolumeId()
	volumePath := req.GetVolumePath()

	if volumePath == "" {
		return &csi.NodeGetVolumeStatsResponse{}, nil
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	device, err := server.getDeviceByFSUUID(volume.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				"on the host to reload",
			"FSUUID", volume.Status.FSUUID)
		client.Eventf(
			volume, corev1.EventTypeWarning, "NodeStageVolume",
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", volume.Status.FSUUID)
		return nil, status.Errorf(codes.NotFound, "unable to find device by FSUUID %v; %v", volume.Status.FSUUID, err)
	}
	quota, err := server.getQuota(ctx, device, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "unable to get quota information; %v", err)
	}

	volUsage := &csi.VolumeUsage{
		Available: volume.Status.TotalCapacity - int64(quota.CurrentSpace),
		Total:     volume.Status.TotalCapacity,
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
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeexpandvolume
func (server *Server) NodeExpandVolume(ctx context.Context, in *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
