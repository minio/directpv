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

package controller

import (
	"context"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/dustin/go-humanize"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

/*  Volume Lifecycle
 *
 *  Creation
 *  -------------
 *   - CreateVolume is called when a PVC is created
 *
 *   - volume is created by finding a drive that satisfies
 *     the storage requirements of the request
 *
 *   - if no such drive is found, then volume creation will
 *     wait until such a drive is made available
 *
 *   - all volumes are created with two finalizers
 *       1. purge-protection: to prevent deletion of volumes
 *          that are in-use (direct.csi.min.io/purge-protection)
 *       2. pv-protection: to prevent deletion of volume while
 *          associated PV is present (direct.csi.min.io/pv-protection)
 *
 *   - a finalizer by the volume name is added to the associated drive
 *
 *   Deletion
 *   --------------
 *   - DeleteVolume is called when the associated PV is deleted and
 *     reclaimPolicy is set to delete
 *
 *   - the PV protection finalizer is first removed on this call
 *
 *   - if deletionTimestamp is set, i.e. the volume resource has
 *     been deleted from the API
 *       - cleanup the mount (not data deletion, just unmounting)
 *
 *       - the finalizer by volume name on the associated drive
 *         is removed by the volume controller
 *
 *       - the purge protection finalizer is first removed by
 *         the volume controller
 *
 */

// Server contains controller server properties
type Server struct{}

// NewServer returns the instance of controller server with the provided properties
func NewServer() *Server {
	return &Server{}
}

// ControllerGetCapabilities constructs ControllerGetCapabilitiesResponse
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#controllergetcapabilities
func (c *Server) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME},
				},
			},
		},
	}, nil
}

// ValidateVolumeCapabilities validates volume capabilities
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#validatevolumecapabilities
func (c *Server) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	var message string
	for _, vcap := range req.GetVolumeCapabilities() {
		if vcap.GetAccessMode() != nil && vcap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			message = fmt.Sprintf("unsupported access mode %s", vcap.GetAccessMode().GetMode())
			break
		}
	}

	response := &csi.ValidateVolumeCapabilitiesResponse{
		Message: message,
	}
	if message == "" {
		response.Confirmed = &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		}
	}

	return response, nil
}

// CreateVolume - Creates a volume
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#createvolume
func (c *Server) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	requiredBytes := int64(-1)
	if req.GetCapacityRange() != nil {
		requiredBytes = req.GetCapacityRange().GetRequiredBytes()
	}
	klog.V(3).InfoS("Create volume requested", "name", req.GetName(), "requiredBytes", humanize.Comma(requiredBytes))

	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "empty volume name in the request")
	}

	for _, vcap := range req.GetVolumeCapabilities() {
		if vcap.GetAccessMode() != nil && vcap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported access mode %s for volume %v", vcap.GetAccessMode().GetMode(), name)
		}
	}

	if len(req.GetVolumeCapabilities()) > 0 && req.GetVolumeCapabilities()[0].GetMount().GetFsType() != "xfs" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported filesystem type %v for volume %v", req.GetVolumeCapabilities()[0].GetMount().GetFsType(), name)
	}

	for key, value := range req.GetParameters() {
		if key == string(directpvtypes.AccessTierLabelKey) {
			if _, err := directpvtypes.StringsToAccessTiers(value); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "unknown access-tier %v for volume %v; %v", value, name, err)
			}
		}
	}

	drive, err := selectDrive(ctx, req)
	if err != nil {
		return nil, err
	}

	klog.V(4).InfoS("Selected drive",
		"drive", drive.GetDriveID(),
		"node", drive.GetNodeID(),
		"name", drive.GetDriveName(),
		"volume", name)

	size := drive.Status.FreeCapacity
	if req.GetCapacityRange() != nil {
		size = req.GetCapacityRange().GetRequiredBytes()
	}

	newVolume := types.NewVolume(
		name,
		drive.Status.FSUUID,
		drive.GetNodeID(),
		drive.GetDriveID(),
		drive.GetDriveName(),
		size,
	)

	if _, err := client.VolumeClient().Create(ctx, newVolume, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.Internal, "unable to create volume %v; %v", name, err)
		}

		volume, err := client.VolumeClient().Get(
			ctx, newVolume.Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		volume.CopyLabels(newVolume)
		volume.Finalizers = newVolume.Finalizers
		volume.Status = newVolume.Status
		_, err = client.VolumeClient().Update(
			ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if err != nil {
			return nil, err
		}

		client.Eventf(volume, client.EventTypeNormal, client.EventReasonVolumeProvisioned, "volume is reused")
	} else {
		client.Eventf(newVolume, client.EventTypeNormal, client.EventReasonVolumeProvisioned, "volume is created")
	}

	if drive.AddVolumeFinalizer(req.GetName()) {
		drive.Status.FreeCapacity -= size
		drive.Status.AllocatedCapacity += size

		klog.V(4).InfoS("Reserving drive",
			"drive", drive.GetDriveID(),
			"node", drive.GetNodeID(),
			"name", drive.GetDriveName(),
			"volume", name)

		_, err = client.DriveClient().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "unable to update reserved drive for volume %v; drive=%v, node=%v, name=%v; %v", name, drive.GetDriveID(), drive.GetNodeID(), drive.GetDriveName(), err)
		}
		client.Eventf(drive, client.EventTypeNormal, client.EventReasonVolumeAdded, "volume %v with size %v is added", name, humanize.Comma(size))
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      name,
			CapacityBytes: size,
			VolumeContext: req.GetParameters(),
			ContentSource: req.GetVolumeContentSource(),
			AccessibleTopology: []*csi.Topology{
				{
					Segments: drive.Status.Topology,
				},
			},
		},
	}, nil
}

// DeleteVolume implements DeleteVolume controller RPC
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#deletevolume
func (c *Server) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(3).InfoS("Delete volume requested", "name", req.GetVolumeId())
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "empty volume ID in the request")
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "unable to get volume %v; %v", volumeID, err)
	}

	if volume.IsStaged() || volume.IsPublished() {
		return nil, status.Errorf(codes.FailedPrecondition, "volume %v is not yet unstaged for deletion", volumeID)
	}

	volume.RemovePVProtection()
	_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to update volume %v; %v", volumeID, err)
	}

	if err = client.VolumeClient().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to delete volume %v; %v", volumeID, err)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

// ListVolumes implements ListVolumes controller RPC
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#listvolumes
func (c *Server) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// ControllerPublishVolume - controller RPC to publish volumes
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#controllerpublishvolume
func (c *Server) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// ControllerUnpublishVolume - controller RPC to unpublish volumes
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#controllerunpublishvolume
func (c *Server) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// ControllerExpandVolume - controller RPC to expand volume
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#controllerexpandvolume
func (c *Server) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// ControllerGetVolume - controller RPC for get volume
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#controllergetvolume
func (c *Server) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// ListSnapshots - controller RPC for listing snapshots
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#listsnapshots
func (c *Server) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// CreateSnapshot controller RPC for creating snapshots
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#createsnapshot
func (c *Server) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// DeleteSnapshot - controller RPC for deleting snapshots
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#deletesnapshot
func (c *Server) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// GetCapacity - controller RPC to get capacity
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#getcapacity
func (c *Server) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
