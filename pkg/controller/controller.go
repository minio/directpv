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

package controller

import (
	"context"
	"fmt"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/matcher"
	"github.com/minio/direct-csi/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type ControllerServer struct {
	NodeID          string
	Identity        string
	Rack            string
	Zone            string
	Region          string
	directcsiClient clientset.Interface
}

func NewControllerServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*ControllerServer, error) {
	controller := &ControllerServer{
		NodeID:          nodeID,
		Identity:        identity,
		Rack:            rack,
		Zone:            zone,
		Region:          region,
		directcsiClient: client.GetDirectClientset(),
	}
	go serveAdmissionController(ctx) // Start admission webhook server
	return controller, nil
}

func (c *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
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

func (c *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
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

// CreateVolume - Creates a DirectCSI Volume
func (c *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	klog.V(3).InfoS("CreateVolumeRequest", "name", req.GetName())
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
		if key == "direct-csi-min-io/access-tier" {
			if _, err := directcsi.ToAccessTier(value); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "unknown access-tier %v; %v", value, err)
			}
		}
	}

	drive, err := selectDrive(ctx, c.directcsiClient.DirectV1beta3().DirectCSIDrives(), req)
	if err != nil {
		return nil, err
	}

	klog.V(4).InfoS("Selected DirectCSI drive",
		"drive-name", drive.Name,
		"node", drive.Status.NodeName,
		"volume", name)

	size := drive.Status.FreeCapacity
	if req.GetCapacityRange() != nil {
		size = req.GetCapacityRange().GetRequiredBytes()
	}

	newVolume := &directcsi.DirectCSIVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				directcsi.DirectCSIVolumeFinalizerPVProtection,
				directcsi.DirectCSIVolumeFinalizerPurgeProtection,
			},
		},
		Status: directcsi.DirectCSIVolumeStatus{
			Drive:             drive.Name,
			NodeName:          drive.Status.NodeName,
			TotalCapacity:     size,
			AvailableCapacity: size,
			UsedCapacity:      0,
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIVolumeConditionStaged),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIVolumeConditionPublished),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directcsi.DirectCSIVolumeReasonNotInUse),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIVolumeConditionReady),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directcsi.DirectCSIVolumeReasonNotReady),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	labels := map[client.LabelKey]client.LabelValue{
		client.NodeLabelKey:      client.NewLabelValue(drive.Status.NodeName),
		client.DrivePathLabelKey: client.NewLabelValue(client.SanitizeDrivePath(drive.Status.Path)),
		client.DriveLabelKey:     client.NewLabelValue(drive.Name),
		client.VersionLabelKey:   directcsi.Version,
		client.CreatedByLabelKey: client.DirectCSIControllerName,
	}
	client.UpdateLabels(newVolume, labels)

	volumeInterface := c.directcsiClient.DirectV1beta3().DirectCSIVolumes()
	if _, err := volumeInterface.Create(ctx, newVolume, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.Internal, "could not create volume %s; %v", name, err)
		}

		volume, err := volumeInterface.Get(
			ctx, newVolume.Name, metav1.GetOptions{TypeMeta: client.DirectCSIVolumeTypeMeta()},
		)
		if err != nil {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		client.SetLabels(volume, labels)
		volume.Finalizers = newVolume.Finalizers
		volume.Status = newVolume.Status
		_, err = volumeInterface.Update(
			ctx, volume, metav1.UpdateOptions{TypeMeta: client.DirectCSIVolumeTypeMeta()},
		)
		if err != nil {
			return nil, err
		}

		client.Eventf(volume, corev1.EventTypeNormal, "VolumeProvisioningSucceeded", "volume %v provisioned", volume.Name)
	} else {
		client.Eventf(newVolume, corev1.EventTypeNormal, "VolumeProvisioningSucceeded", "volume %v is created", newVolume.Name)
	}

	finalizer := directcsi.DirectCSIDriveFinalizerPrefix + req.GetName()
	if !matcher.StringIn(drive.Finalizers, finalizer) {
		drive.Status.FreeCapacity = drive.Status.FreeCapacity - size
		drive.Status.AllocatedCapacity = drive.Status.AllocatedCapacity + size
		drive.Status.DriveStatus = directcsi.DriveStatusInUse
		drive.SetFinalizers(append(drive.GetFinalizers(), finalizer))

		klog.V(4).InfoS("Reserving DirectCSI drive",
			"drive-name", drive.Name,
			"node", drive.Status.NodeName,
			"volume", name)

		_, err = c.directcsiClient.DirectV1beta3().DirectCSIDrives().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: client.DirectCSIDriveTypeMeta()},
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not reserve drive[%s] %v", drive.Name, err)
		} else {
			client.Eventf(drive, corev1.EventTypeNormal, "DriveReservationSucceded", "reserved drive %v on node %v and volume %v", drive.Name, drive.Status.NodeName, name)
		}
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

func (c *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	klog.V(3).InfoS("DeleteVolumeRequest", "name", req.GetVolumeId())
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	directCSIClient := c.directcsiClient.DirectV1beta3()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.NotFound, "could not retreive volume [%s]: %v", vID, err)
	}

	// Do not proceed if the volume hasn't been unpublished or unstaged
	if utils.IsConditionStatus(vol.Status.Conditions,
		string(directcsi.DirectCSIVolumeConditionStaged),
		metav1.ConditionTrue) ||
		utils.IsConditionStatus(vol.Status.Conditions,
			string(directcsi.DirectCSIVolumeConditionPublished),
			metav1.ConditionTrue) {
		return nil, status.Errorf(codes.FailedPrecondition,
			"waiting for volume [%s] to be unstaged before deleting", vID)
	}

	finalizers := vol.GetFinalizers()
	updatedFinalizers := []string{}
	for _, f := range finalizers {
		if f == directcsi.DirectCSIVolumeFinalizerPVProtection {
			continue
		}
		updatedFinalizers = append(updatedFinalizers, f)
	}
	vol.SetFinalizers(updatedFinalizers)

	_, err = vclient.Update(ctx, vol, metav1.UpdateOptions{
		TypeMeta: client.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not remove finalizer for volume [%s]: %v", vID, err)
	}

	if err = vclient.Delete(ctx, vol.Name, metav1.DeleteOptions{}); err != nil {
		return nil, status.Errorf(codes.Internal, "could not delete volume [%s]: %v", vID, err)
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (c *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) ControllerGetVolume(ctx context.Context, req *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (c *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
