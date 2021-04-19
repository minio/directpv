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
	"sort"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FilterDrivesByVolumeRequest - Filters the CSI drives by create volume request
func FilterDrivesByVolumeRequest(volReq *csi.CreateVolumeRequest, csiDrives []directcsi.DirectCSIDrive) ([]directcsi.DirectCSIDrive, error) {
	capacityRange := volReq.GetCapacityRange()
	vCaps := volReq.GetVolumeCapabilities()
	fsType := ""
	if len(vCaps) > 0 {
		fsType = vCaps[0].GetMount().GetFsType()
	}

	filteredDrivesByFormat := FilterDrivesByRequestFormat(csiDrives)
	if len(filteredDrivesByFormat) == 0 {
		return []directcsi.DirectCSIDrive{}, status.Error(codes.FailedPrecondition, "No csi drives are been added. Please use `add drives` plugin command to add the drives")
	}

	capFilteredDrives := FilterDrivesByCapacityRange(capacityRange, filteredDrivesByFormat)
	if len(capFilteredDrives) == 0 {
		return []directcsi.DirectCSIDrive{}, status.Error(codes.OutOfRange, "Invalid capacity range")
	}

	fsFilteredDrives := FilterDrivesByFsType(fsType, capFilteredDrives)
	if len(fsFilteredDrives) == 0 {
		return []directcsi.DirectCSIDrive{}, status.Errorf(codes.InvalidArgument, "Cannot find any drives by the fstype: %s", fsType)
	}

	paramFilteredDrives, pErr := FilterDrivesByParameters(volReq.GetParameters(), fsFilteredDrives)
	if pErr != nil {
		return fsFilteredDrives, status.Errorf(codes.InvalidArgument, "Error while filtering based on sc parameters: %v", pErr)
	}
	if len(paramFilteredDrives) == 0 {
		return []directcsi.DirectCSIDrive{}, status.Errorf(codes.InvalidArgument, "Cannot match any drives by the provided storage class parameters: %s", volReq.GetParameters())
	}

	return paramFilteredDrives, nil
}

// FilterDrivesByCapacityRange - Filters the CSI drives by capacity range in the create volume request
func FilterDrivesByCapacityRange(capacityRange *csi.CapacityRange, csiDrives []directcsi.DirectCSIDrive) []directcsi.DirectCSIDrive {
	reqBytes := capacityRange.GetRequiredBytes()
	//limitBytes := capacityRange.GetLimitBytes()
	filteredDriveList := []directcsi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.FreeCapacity >= reqBytes {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByRequestFormat - Selects the drives only if the requested format is empty/satisfied already.
func FilterDrivesByRequestFormat(csiDrives []directcsi.DirectCSIDrive) []directcsi.DirectCSIDrive {
	filteredDriveList := []directcsi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		dStatus := csiDrive.Status.DriveStatus
		if dStatus == directcsi.DriveStatusReady ||
			dStatus == directcsi.DriveStatusInUse {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByFsType - Filters the CSI drives by filesystem
func FilterDrivesByFsType(fsType string, csiDrives []directcsi.DirectCSIDrive) []directcsi.DirectCSIDrive {
	if fsType == "" {
		return csiDrives
	}
	filteredDriveList := []directcsi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.Filesystem == fsType {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByParameters - Filters the CSI drives by request parameters
func FilterDrivesByParameters(parameters map[string]string, csiDrives []directcsi.DirectCSIDrive) ([]directcsi.DirectCSIDrive, error) {
	filteredDriveList := csiDrives
	for k, v := range parameters {
		switch k {
		case "direct-csi-min-io/access-tier":
			accessT, err := utils.ValidateAccessTier(v)
			if err != nil {
				return csiDrives, err
			}
			filteredDriveList = FilterDrivesByAccessTier(accessT, filteredDriveList)
		default:
		}
	}
	return filteredDriveList, nil
}

func FilterDrivesByAccessTier(accessTier directcsi.AccessTier, csiDrives []directcsi.DirectCSIDrive) []directcsi.DirectCSIDrive {
	filteredDriveList := []directcsi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.AccessTier == accessTier {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByTopologyRequirements - selects the CSI drive by topology in the create volume request
func FilterDrivesByTopologyRequirements(volReq *csi.CreateVolumeRequest, csiDrives []directcsi.DirectCSIDrive) (directcsi.DirectCSIDrive, error) {
	tReq := volReq.GetAccessibilityRequirements()

	preferredXs := tReq.GetPreferred()
	requisiteXs := tReq.GetRequisite()

	// Sort the drives by free capacity [Descending]
	sort.SliceStable(csiDrives, func(i, j int) bool {
		return csiDrives[i].Status.FreeCapacity > csiDrives[j].Status.FreeCapacity
	})

	// Try to fullfill the preferred topology request, If not, fallback to requisite list.
	// Ref: https://godoc.org/github.com/container-storage-interface/spec/lib/go/csi#TopologyRequirement
	for _, preferredTop := range preferredXs {
		if selectedDrive, err := selectDriveByTopology(preferredTop, csiDrives); err == nil {
			return selectedDrive, nil
		}
	}

	for _, requisiteTop := range requisiteXs {
		if selectedDrive, err := selectDriveByTopology(requisiteTop, csiDrives); err == nil {
			return selectedDrive, nil
		}
	}

	if len(preferredXs) == 0 && len(requisiteXs) == 0 {
		return csiDrives[0], nil
	}

	return directcsi.DirectCSIDrive{}, status.Error(codes.ResourceExhausted, "Cannot satisfy the topology constraint")
}

func selectDriveByTopology(top *csi.Topology, csiDrives []directcsi.DirectCSIDrive) (directcsi.DirectCSIDrive, error) {
	topSegments := top.GetSegments()
	for _, csiDrive := range csiDrives {
		driveSegments := csiDrive.Status.Topology
		if matchSegments(topSegments, driveSegments) {
			return csiDrive, nil
		}
	}
	return directcsi.DirectCSIDrive{}, status.Error(codes.ResourceExhausted, "Cannot satisfy the topology constraint")
}

func matchSegments(topSegments, driveSegments map[string]string) bool {
	req := len(topSegments)
	match := 0
	for k, v := range topSegments {
		if dval, ok := driveSegments[k]; ok && dval == v {
			match = match + 1
		} else {
			break
		}
	}
	return req == match
}
