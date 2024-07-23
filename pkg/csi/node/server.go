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

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/metrics"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// Server denotes node server.
type Server struct {
	csi.UnimplementedNodeServer

	nodeID   directpvtypes.NodeID
	identity string
	rack     string
	zone     string
	region   string

	getMounts         func() (mountMap, rootMap map[string]utils.StringSet, err error)
	getDeviceByFSUUID func(fsuuid string) (string, error)
	bindMount         func(source, target string, readOnly bool) error
	unmount           func(target string) error
	getQuota          func(ctx context.Context, device, volumeName string) (quota *xfs.Quota, err error)
	setQuota          func(ctx context.Context, device, path, volumeName string, quota xfs.Quota, update bool) (err error)
	mkdir             func(path string) error
}

func newServer(identity string, nodeID directpvtypes.NodeID, rack, zone, region string) Server {
	return Server{
		nodeID:   nodeID,
		identity: identity,
		rack:     rack,
		zone:     zone,
		region:   region,

		getMounts: func() (mountMap, rootMap map[string]utils.StringSet, err error) {
			mountMap, _, _, rootMap, err = sys.GetMounts(false)
			return
		},
		getDeviceByFSUUID: sys.GetDeviceByFSUUID,
		bindMount:         xfs.BindMount,
		unmount:           func(target string) error { return sys.Unmount(target, true, true, false) },
		getQuota:          xfs.GetQuota,
		setQuota:          xfs.SetQuota,
		mkdir: func(dir string) error {
			return sys.Mkdir(dir, 0o755)
		},
	}
}

// NewServer creates node server.
func NewServer(ctx context.Context,
	identity string, nodeID directpvtypes.NodeID, rack, zone, region string,
	metricsPort int,
) *Server {
	go metrics.ServeMetrics(ctx, nodeID, metricsPort)
	server := newServer(identity, nodeID, rack, zone, region)
	return &server
}

// NodeGetInfo gets node information.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetinfo
func (server *Server) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			string(directpvtypes.TopologyDriverIdentity): server.identity,
			string(directpvtypes.TopologyDriverRack):     server.rack,
			string(directpvtypes.TopologyDriverZone):     server.zone,
			string(directpvtypes.TopologyDriverRegion):   server.region,
			string(directpvtypes.TopologyDriverNode):     string(server.nodeID),
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             string(server.nodeID),
		AccessibleTopology: topology,
	}, nil
}

// NodeGetCapabilities gets node capabilities.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetcapabilities
func (server *Server) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
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
			nodeCap(csi.NodeServiceCapability_RPC_EXPAND_VOLUME),
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
			volume, client.EventTypeWarning, client.EventReasonMetrics,
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

// NodeExpandVolume handles expand volume request.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodeexpandvolume
func (server *Server) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	requiredBytes := int64(-1)
	if req.CapacityRange != nil {
		requiredBytes = req.CapacityRange.RequiredBytes
	}
	klog.V(3).InfoS("Expand volume requested",
		"volumeID", req.GetVolumeId(),
		"VolumePath", req.GetVolumePath(),
		"requiredBytes", requiredBytes,
	)

	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}
	if req.GetVolumePath() == "" {
		return nil, status.Error(codes.InvalidArgument, "volumePath missing in request")
	}
	if requiredBytes == -1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid capacity range in the request for volume %v expansion", volumeID)
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		code := codes.Internal
		if errors.IsNotFound(err) {
			code = codes.NotFound
		}
		return nil, status.Errorf(code, "unable to get volume %v; %v", volumeID, err)
	}

	if !volume.IsStaged() {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %v is not yet staged, but requested for volume expansion", volume.Name)
	}

	if volume.Status.TotalCapacity >= requiredBytes {
		// As per the specification, nothing to do
		return &csi.NodeExpandVolumeResponse{CapacityBytes: requiredBytes}, nil
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
			volume, client.EventTypeWarning, client.EventReasonStageVolume,
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", volume.Status.FSUUID)
		return nil, status.Errorf(codes.Internal, "unable to find device by FSUUID %v; %v", volume.Status.FSUUID, err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(requiredBytes),
		SoftLimit: uint64(requiredBytes),
	}

	if err := server.setQuota(ctx, device, volume.Status.DataPath, volume.Name, quota, true); err != nil {
		klog.ErrorS(err, "unable to set quota on volume data path", "DataPath", volume.Status.DataPath)
		return nil, status.Errorf(codes.Internal, "unable to set quota on volume data path; %v", err)
	}

	volume.Status.TotalCapacity = requiredBytes
	volume.Status.AvailableCapacity = volume.Status.TotalCapacity - volume.Status.UsedCapacity
	_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to update volume %v; %v", volumeID, err)
	}

	return &csi.NodeExpandVolumeResponse{CapacityBytes: requiredBytes}, nil
}
