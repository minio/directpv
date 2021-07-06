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
	"path/filepath"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func NewControllerServer(ctx context.Context, identity, nodeID, rack, zone, region string) (*ControllerServer, error) {
	// Start admission webhook server
	go serveAdmissionController(ctx)

	kubeConfig := utils.GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return &ControllerServer{}, err
		}
	}

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return &ControllerServer{}, err
	}

	return &ControllerServer{
		NodeID:          nodeID,
		Identity:        identity,
		Rack:            rack,
		Zone:            zone,
		Region:          region,
		directcsiClient: directClientset,
	}, nil
}

type ControllerServer struct {
	NodeID          string
	Identity        string
	Rack            string
	Zone            string
	Region          string
	directcsiClient clientset.Interface
}

func (c *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	controllerCap := func(cap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
		klog.V(4).Infof("Using controller capability %v", cap)

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
	klog.V(4).Infof("CreateVolumeRequest: %v", req)
	name := req.GetName()
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "volume name cannot be empty")
	}

	directCSIClient := c.directcsiClient.DirectV1beta2()
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

	matchDrive := func() (*directcsi.DirectCSIDrive, error) {
		driveList, err := dclient.List(ctx, metav1.ListOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "could not retreive directcsidrives: %v", err)
		}
		drives := driveList.Items

		volFinalizer := directcsi.DirectCSIDriveFinalizerPrefix + name
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

		selectedDrive, err := FilterDrivesByTopologyRequirements(req, filteredDrives)
		if err != nil {
			return nil, err
		}
		klog.V(4).Infof("Selected DirectCSI drive: (Name: %s, NodeName: %s)", selectedDrive.Name, selectedDrive.Status.NodeName)

		return &selectedDrive, nil
	}

	getSize := func(drive *directcsi.DirectCSIDrive) int64 {
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

	reserveDrive := func(drive *directcsi.DirectCSIDrive, size int64) error {
		var alreadyReserved bool
		finalizer := directcsi.DirectCSIDriveFinalizerPrefix + name
		finalizers := drive.GetFinalizers()
		for _, f := range finalizers {
			if f == finalizer {
				alreadyReserved = true
				break
			}
		}
		// only if drive was not previously reserved
		if !alreadyReserved {
			drive.Status.FreeCapacity = drive.Status.FreeCapacity - size
			drive.Status.AllocatedCapacity = drive.Status.AllocatedCapacity + size
			if drive.Status.DriveStatus == directcsi.DriveStatusReady {
				drive.Status.DriveStatus = directcsi.DriveStatusInUse
			}

			finalizers = append(finalizers, finalizer)
			drive.SetFinalizers(finalizers)

			klog.V(4).Infof("Reserving DirectCSI drive: (Name: %s, NodeName: %s)", drive.Name, drive.Status.NodeName)
			if _, err := dclient.Update(ctx, drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
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
	vol := &directcsi.DirectCSIVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				string(directcsi.DirectCSIVolumeFinalizerPVProtection),
				string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
			},
			Labels: map[string]string{
				directcsi.Group + "/node":       drive.Status.NodeName,
				directcsi.Group + "/drive-path": filepath.Base(drive.Status.Path),
				directcsi.Group + "/drive":      utils.SanitizeLabelV(drive.Name),
				directcsi.Group + "/version":    directcsi.Version,
				directcsi.Group + "/created-by": "directcsi-controller",
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

	if _, err := vclient.Create(ctx, vol, metav1.CreateOptions{}); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, status.Errorf(codes.Internal, "could not create volume [%s]: %v", name, err)
		}
		existingVol, gErr := vclient.Get(ctx, vol.Name, metav1.GetOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		})
		if gErr != nil {
			return nil, status.Error(codes.NotFound, gErr.Error())
		}
		existingVol.ObjectMeta.Finalizers = vol.ObjectMeta.Finalizers
		existingVol.Status = vol.Status
		if _, cErr := vclient.Update(ctx, existingVol, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		}); cErr != nil {
			return nil, cErr
		}
	}

	if err := reserveDrive(drive, size); err != nil {
		return nil, err
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

	directCSIClient := c.directcsiClient.DirectV1beta2()
	vclient := directCSIClient.DirectCSIVolumes()

	vol, err := vclient.Get(ctx, vID, metav1.GetOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return &csi.DeleteVolumeResponse{}, nil
		}
		return nil, status.Errorf(codes.NotFound, "could not retreive volume [%s]: %v", vID, err)
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
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
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
