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
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
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
type Server struct {
	NodeID   string
	Identity string
	Rack     string
	Zone     string
	Region   string
}

// NewServer returns the instance of controller server with the provided properties
func NewServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*Server, error) {
	controller := &Server{
		NodeID:   nodeID,
		Identity: identity,
		Rack:     rack,
		Zone:     zone,
		Region:   region,
	}
	return controller, nil
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
	klog.V(3).InfoS("CreateVolume() called", "name", req.GetName())
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name cannot be empty")
	}

	for _, vcap := range req.GetVolumeCapabilities() {
		if vcap.GetAccessMode() != nil && vcap.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported access mode: %s", vcap.GetAccessMode().GetMode())
		}
	}

	if len(req.GetVolumeCapabilities()) > 0 && req.GetVolumeCapabilities()[0].GetMount().GetFsType() != "xfs" {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported filesystem type %v", req.GetVolumeCapabilities()[0].GetMount().GetFsType())
	}

	for key, value := range req.GetParameters() {
		if key == string(types.AccessTierLabelKey) {
			if _, err := directpvtypes.StringsToAccessTiers(value); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "unknown access-tier %v; %v", value, err)
			}
		}
	}

	drive, err := selectDrive(ctx, req)
	if err != nil {
		return nil, err
	}

	klog.V(4).InfoS("Selected drive",
		"drive-name", drive.Name,
		"node", drive.Status.NodeName,
		"volume", name)

	size := drive.Status.FreeCapacity
	if req.GetCapacityRange() != nil {
		size = req.GetCapacityRange().GetRequiredBytes()
	}

	newVolume := &types.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				consts.VolumeFinalizerPVProtection,
				consts.VolumeFinalizerPurgeProtection,
			},
		},
		Status: types.VolumeStatus{
			DriveName:         drive.Name,
			FSUUID:            drive.Status.FSUUID,
			NodeName:          drive.Status.NodeName,
			TotalCapacity:     size,
			AvailableCapacity: size,
		},
	}

	labels := map[types.LabelKey]types.LabelValue{
		types.NodeLabelKey:      types.NewLabelValue(drive.Status.NodeName),
		types.DrivePathLabelKey: types.NewLabelValue(drive.Status.Path),
		types.DriveLabelKey:     types.NewLabelValue(drive.Name),
		types.VersionLabelKey:   types.NewLabelValue(consts.LatestAPIVersion),
		types.CreatedByLabelKey: types.NewLabelValue(consts.ControllerName),
	}
	types.UpdateLabels(newVolume, labels)

	if _, err := client.VolumeClient().Create(ctx, newVolume, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.Internal, "could not create volume %s; %v", name, err)
		}

		volume, err := client.VolumeClient().Get(
			ctx, newVolume.Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		types.SetLabels(volume, labels)
		volume.Finalizers = newVolume.Finalizers
		volume.Status = newVolume.Status
		_, err = client.VolumeClient().Update(
			ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
		)
		if err != nil {
			return nil, err
		}

		client.Eventf(volume, corev1.EventTypeNormal, "VolumeProvisioningSucceeded", "volume %v provisioned", volume.Name)
	} else {
		client.Eventf(newVolume, corev1.EventTypeNormal, "VolumeProvisioningSucceeded", "volume %v is created", newVolume.Name)
	}

	finalizer := consts.DriveFinalizerPrefix + req.GetName()
	if !utils.ItemIn(drive.Finalizers, finalizer) {
		drive.Status.FreeCapacity -= size
		drive.Status.AllocatedCapacity += size
		drive.SetFinalizers(append(drive.GetFinalizers(), finalizer))

		klog.V(4).InfoS("Reserving drive",
			"drive-name", drive.Name,
			"node", drive.Status.NodeName,
			"volume", name)

		_, err = client.DriveClient().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not reserve drive[%s] %v", drive.Name, err)
		}
		client.Eventf(drive, corev1.EventTypeNormal, "DriveReservationSucceded", "reserved drive %v on node %v and volume %v", drive.Name, drive.Status.NodeName, name)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      name,
			CapacityBytes: int64(size),
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
	klog.V(3).InfoS("DeleteVolume() called", "name", req.GetVolumeId())
	volumeID := req.GetVolumeId()
	if volumeID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.NotFound, "could not retrieve volume [%s]: %v", volumeID, err)
	}

	if volume.Status.IsStaged() || volume.Status.IsPublished() {
		return nil, status.Errorf(codes.FailedPrecondition,
			"waiting for volume [%s] to be unstaged before deleting", volumeID)
	}

	finalizers := volume.GetFinalizers()
	updatedFinalizers := []string{}
	for _, f := range finalizers {
		if f == consts.VolumeFinalizerPVProtection {
			continue
		}
		updatedFinalizers = append(updatedFinalizers, f)
	}
	volume.SetFinalizers(updatedFinalizers)

	_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
		TypeMeta: types.NewVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not remove finalizer for volume [%s]: %v", volumeID, err)
	}

	if err = client.VolumeClient().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil {
		return nil, status.Errorf(codes.Internal, "could not delete volume [%s]: %v", volumeID, err)
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
