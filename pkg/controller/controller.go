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
	"fmt"
	"strconv"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func NewControllerServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*ControllerServer, error) {

	// Start DirectCSI volume listener
	go startVolumeController(ctx, nodeID)

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
	glog.V(5).Infof("ControllerGetCapabilities: called with args %+v", *req)
	volCaps := req.GetVolumeCapabilities()

	confirmed := &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
		VolumeCapabilities: volCaps,
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: confirmed,
	}, nil
}

// CreateVolume - Creates a DirectCSI Volume
func (c *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name cannot be empty")
	}
	glog.V(5).Infof("CreateVolumeRequest - %s", name)

	vc := req.GetVolumeCapabilities()
	if vc == nil || len(vc) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume capabilities cannot be empty")
	}

	accessModeWrapper := vc[0].GetAccessMode()
	var accessMode csi.VolumeCapability_AccessMode_Mode
	if accessModeWrapper != nil {
		accessMode = accessModeWrapper.GetMode()
		if accessMode != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return nil, status.Errorf(codes.InvalidArgument, "unsupported access mode: %s", accessModeWrapper.String())
		}
	}

	var selectedCSIDrive direct_csi.DirectCSIDrive
	var vol *direct_csi.DirectCSIVolume
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		directCSIClient := utils.GetDirectCSIClient()
		driveList, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
		if err != nil || len(driveList.Items) == 0 {
			return status.Error(codes.NotFound, err.Error())
		}
		directCSIDrives := driveList.Items

		filteredDrives, fErr := FilterDrivesByVolumeRequest(req, directCSIDrives)
		if fErr != nil {
			return fErr
		}

		var dErr error
		selectedCSIDrive, dErr = SelectDriveByTopologyReq(req.GetAccessibilityRequirements(), filteredDrives)
		if dErr != nil {
			return dErr
		}

		totalVolumeCapacity := req.GetCapacityRange().GetRequiredBytes()
		if totalVolumeCapacity == int64(0) {
			totalVolumeCapacity = selectedCSIDrive.Status.TotalCapacity
		}

		conditions := []metav1.Condition{
			{Type: "staged", Status: metav1.ConditionFalse, LastTransitionTime: metav1.Now(), Reason: "VolumeStaged", Message: "VolumeStaged"},
			{Type: "published", Status: metav1.ConditionFalse, LastTransitionTime: metav1.Now(), Reason: "VolumePublished", Message: "VolumePublished"},
			{Type: "volumestats", Status: metav1.ConditionTrue, LastTransitionTime: metav1.Now(), Reason: "StatsRefreshed", Message: "StatsRefreshed"},
		}

		vol = &direct_csi.DirectCSIVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Finalizers: []string{fmt.Sprintf("%s/%s", direct_csi.SchemeGroupVersion.Group, "pv-protection"), fmt.Sprintf("%s/%s", direct_csi.SchemeGroupVersion.Group, "purge-protection")},
			},
			Status: direct_csi.DirectCSIVolumeStatus{
				OwnerDrive:        selectedCSIDrive.ObjectMeta.Name,
				OwnerNode:         selectedCSIDrive.Status.NodeName,
				TotalCapacity:     totalVolumeCapacity,
				AvailableCapacity: totalVolumeCapacity,
				UsedCapacity:      int64(0),
				Conditions:        conditions,
			},
		}

		var uErr error
		if _, err = directCSIClient.DirectCSIVolumes().Create(ctx, vol, metav1.CreateOptions{}); err != nil {
			if errors.IsAlreadyExists(err) {
				existingVol, vErr := directCSIClient.DirectCSIVolumes().Get(ctx, vol.ObjectMeta.Name, metav1.GetOptions{})
				if vErr != nil {
					return vErr
				}
				existingVol.Status.OwnerDrive = selectedCSIDrive.ObjectMeta.Name
				existingVol.Status.OwnerNode = selectedCSIDrive.Status.NodeName
				existingVol.Status.TotalCapacity = totalVolumeCapacity
				existingVol.Status.AvailableCapacity = totalVolumeCapacity
				existingVol.Status.UsedCapacity = int64(0)
				existingVol.Status.Conditions = conditions
				vol, uErr = directCSIClient.DirectCSIVolumes().Update(ctx, existingVol, metav1.UpdateOptions{})
				if uErr != nil {
					return uErr
				}
			} else {
				return err
			}
		}

		glog.Infof("Created DirectCSI Volume - %s", vol.ObjectMeta.Name)

		copiedDrive := selectedCSIDrive.DeepCopy()
		copiedDrive.Status.FreeCapacity = copiedDrive.Status.FreeCapacity - req.GetCapacityRange().GetRequiredBytes()
		copiedDrive.Status.AllocatedCapacity = copiedDrive.Status.AllocatedCapacity + req.GetCapacityRange().GetRequiredBytes()
		copiedDrive.Status.DriveStatus = direct_csi.InUse
		copiedDrive.ObjectMeta.SetFinalizers(utils.AddFinalizer(&copiedDrive.ObjectMeta, fmt.Sprintf("%s/%s", direct_csi.SchemeGroupVersion.Group, vol.ObjectMeta.Name)))
		if _, err := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{}); err != nil {
			return err
		}
		glog.Infof("Updated DirectCSIDrive - %s", copiedDrive.Name)

		return nil
	})
	if err != nil {
		return nil, err
	}

	volumeContext := req.GetParameters()
	volumeContext["RequiredBytes"] = strconv.FormatInt(req.GetCapacityRange().GetRequiredBytes(), 10)

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      vol.ObjectMeta.Name,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			VolumeContext: req.GetParameters(),
			ContentSource: req.GetVolumeContentSource(),
			AccessibleTopology: []*csi.Topology{
				{
					Segments: selectedCSIDrive.Status.Topology,
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
	glog.V(5).Infof("DeleteVolumeRequest - %s", vID)

	directCSIClient := utils.GetDirectCSIClient()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dvol, cErr := directCSIClient.DirectCSIVolumes().Get(ctx, vID, metav1.GetOptions{})
		if cErr != nil {
			return cErr
		}
		if !utils.CheckVolumeStatusCondition(dvol.Status.Conditions, "staged", metav1.ConditionFalse) {
			return status.Errorf(codes.FailedPrecondition, "volume has not been unstaged: %s", vID)
		}
		dvol.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&dvol.ObjectMeta, fmt.Sprintf("%s/%s", direct_csi.SchemeGroupVersion.Group, "pv-protection")))
		if dvol, cErr = directCSIClient.DirectCSIVolumes().Update(ctx, dvol, metav1.UpdateOptions{}); cErr != nil {
			return cErr
		}
		return nil
	}); err != nil {
		if errors.IsNotFound(err) || errors.IsGone(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, err
	}

	// Uncomment the following to test the purge-protection flow.
	// if dErr := directCSIClient.DirectCSIVolumes().Delete(ctx, vID, metav1.DeleteOptions{}); dErr != nil {
	// 	return nil, status.Error(codes.Internal, dErr.Error())
	// }

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
