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

package controller

import (
	"context"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
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

func NewControllerServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*ControllerServer, error) {
	// Start admission webhook server
	go serveAdmissionController(ctx)

	return &ControllerServer{
		NodeID:   nodeID,
		Identity: identity,
		Rack:     rack,
		Zone:     zone,
		Region:   region,
	}, nil
}

type ControllerServer struct {
	NodeID   string
	Identity string
	Rack     string
	Zone     string
	Region   string
}

func (c *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	controllerCap := func(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
		glog.Infof("Using controller capability %v", cap)

		return &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: cap,
				},
			},
		}
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			controllerCap(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME),
		},
	}, nil
}

func (c *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	validateVolumeCapabilities := func() error {
		vcaps := req.GetVolumeCapabilities()
		for _, vcap := range vcaps {
			access := vcap.GetAccessMode()
			if access != nil {
				mode := access.GetMode()
				if mode != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
					return status.Errorf(codes.InvalidArgument, "unsupported access mode: %s", mode)
				}
			}
		}
		return nil
	}

	var confirmed *csi.ValidateVolumeCapabilitiesResponse_Confirmed
	message := ""
	if err := validateVolumeCapabilities(); err != nil {
		confirmed = &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeCapabilities: req.GetVolumeCapabilities(),
		}
	} else {
		message = err.Error()
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: confirmed,
		Message:   message,
	}, nil
}

// CreateVolume - Creates a DirectCSI Volume
func (c *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name cannot be empty")
	}

	directCSIClient := utils.GetDirectCSIClient()
	dclient := directCSIClient.DirectCSIDrives()
	vclient := directCSIClient.DirectCSIVolumes()

	validateVolumeCapabilities := func() error {
		vcaps := req.GetVolumeCapabilities()
		for _, vcap := range vcaps {
			access := vcap.GetAccessMode()
			if access != nil {
				mode := access.GetMode()
				if mode != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
					return status.Errorf(codes.InvalidArgument, "unsupported access mode: %s", mode)
				}
			}
		}
		return nil
	}

	matchDrive := func() (*directv1alpha1.DirectCSIDrive, error) {
		driveList, err := dclient.List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "could not retreive directcsidrives: %v", err)
		}
		drives := driveList.Items

		volFinalizer := directv1alpha1.DirectCSIDriveFinalizerPrefix + name
		for _, drive := range drives {
			finalizers := drive.GetFinalizers()
			for _, f := range finalizers {
				if f == volFinalizer {
					return &drive, nil
				}
			}
		}

		filteredDrives, err := FilterDrivesByVolumeRequest(req, drives)
		if err != nil {
			return nil, err
		}

		utils.JSONifyAndLog(req.GetAccessibilityRequirements())

		selectedDrive, err := FilterDrivesByTopologyRequirements(req, filteredDrives)
		if err != nil {
			return nil, err
		}
		return &selectedDrive, nil
	}

	getSize := func(drive *directv1alpha1.DirectCSIDrive) int64 {
		var size int64
		if capRange := req.GetCapacityRange(); capRange != nil {
			size = capRange.GetRequiredBytes()
		}
		// if no size requirement is specified, occupy all the free capacity on the drive
		if size == 0 {
			size = drive.Status.FreeCapacity
		}
		return size
	}

	reserveDrive := func(drive *directv1alpha1.DirectCSIDrive) error {
		size := getSize(drive)

		var found bool
		finalizer := directv1alpha1.DirectCSIDriveFinalizerPrefix + name
		finalizers := drive.GetFinalizers()
		for _, f := range finalizers {
			if f == finalizer {
				found = true
				break
			}
		}
		// only if drive was not previously reserved
		if !found {
			drive.Status.FreeCapacity = drive.Status.FreeCapacity - size
			drive.Status.AllocatedCapacity = drive.Status.AllocatedCapacity + size
			if drive.Status.DriveStatus == directv1alpha1.DriveStatusReady {
				drive.Status.DriveStatus = directv1alpha1.DriveStatusInUse
			}

			finalizers = append(finalizers, finalizer)
			drive.SetFinalizers(finalizers)

			if _, err := dclient.Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
				return status.Errorf(codes.Internal, "could not reserve drive[%s] %v", drive.Name, err)
			}
		}

		return nil
	}

	if err := validateVolumeCapabilities(); err != nil {
		return nil, err
	}

	drive, err := matchDrive()
	if err != nil {
		return nil, err
	}

	size := getSize(drive)

	// reserve the drive first, then create the volume
	if err := reserveDrive(drive); err != nil {
		return nil, err
	}

	vol := &directv1alpha1.DirectCSIVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				string(directv1alpha1.DirectCSIVolumeFinalizerPVProtection),
				string(directv1alpha1.DirectCSIVolumeFinalizerPurgeProtection),
			},
		},
		Status: directv1alpha1.DirectCSIVolumeStatus{
			Drive:             drive.Name,
			NodeName:          drive.Status.NodeName,
			TotalCapacity:     size,
			AvailableCapacity: size,
			UsedCapacity:      0,
			Conditions: []metav1.Condition{
				{
					Type:               string(directv1alpha1.DirectCSIVolumeConditionStaged),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directv1alpha1.DirectCSIVolumeReasonNotInUse),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directv1alpha1.DirectCSIVolumeConditionPublished),
					Status:             metav1.ConditionFalse,
					Message:            "",
					Reason:             string(directv1alpha1.DirectCSIVolumeReasonNotInUse),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	if _, err = vclient.Create(ctx, vol, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.Internal, "could not create volume [%s]: %v", name, err)
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
	vID := req.GetVolumeId()
	if vID == "" {
		return nil, status.Error(codes.InvalidArgument, "volume ID missing in request")
	}

	directCSIClient := utils.GetDirectCSIClient()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.NotFound, "could not retreive volume [%s]: %v", vID, err)
	}

	finalizers := vol.GetFinalizers()
	updatedFinalizers := []string{}
	for _, f := range finalizers {
		if f == directv1alpha1.DirectCSIVolumeFinalizerPVProtection {
			continue
		}
		updatedFinalizers = append(updatedFinalizers, f)
	}
	vol.SetFinalizers(updatedFinalizers)

	_, err = vclient.Update(ctx, vol, metav1.UpdateOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not remove finalizer for volume [%s]: %v", vID, err)
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
