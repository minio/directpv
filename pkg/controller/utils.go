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
	"reflect"
	"sort"

	"github.com/container-storage-interface/spec/lib/go/csi"

	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FilterDrivesByVolumeRequest - Filters the CSI drives by create volume request
func FilterDrivesByVolumeRequest(volReq *csi.CreateVolumeRequest, csiDrives []direct_csi.DirectCSIDrive) ([]direct_csi.DirectCSIDrive, error) {
	capacityRange := volReq.GetCapacityRange()
	vCaps := volReq.GetVolumeCapabilities()
	fsType := ""
	if len(vCaps) > 0 {
		fsType = vCaps[0].GetMount().GetFsType()
	}

	filteredDrivesByFormat := FilterDrivesByRequestFormat(csiDrives)
	if len(filteredDrivesByFormat) == 0 {
		return []direct_csi.DirectCSIDrive{}, status.Error(codes.FailedPrecondition, "No csi drives are been added. Please use `add drives` plugin command to add the drives")
	}

	capFilteredDrives := FilterDrivesByCapacityRange(capacityRange, filteredDrivesByFormat)
	if len(capFilteredDrives) == 0 {
		return []direct_csi.DirectCSIDrive{}, status.Error(codes.OutOfRange, "Invalid capacity range")
	}

	fsFilteredDrives := FilterDrivesByFsType(fsType, capFilteredDrives)
	if len(fsFilteredDrives) == 0 {
		return []direct_csi.DirectCSIDrive{}, status.Errorf(codes.InvalidArgument, "Cannot find any drives by the fstype: %s", fsType)
	}

	return fsFilteredDrives, nil
}

// FilterDrivesByCapacityRange - Filters the CSI drives by capacity range in the create volume request
func FilterDrivesByCapacityRange(capacityRange *csi.CapacityRange, csiDrives []direct_csi.DirectCSIDrive) []direct_csi.DirectCSIDrive {
	reqBytes := capacityRange.GetRequiredBytes()
	limitBytes := capacityRange.GetLimitBytes()
	filteredDriveList := []direct_csi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.FreeCapacity >= reqBytes && (limitBytes == 0 || csiDrive.Status.FreeCapacity <= limitBytes) {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByRequestFormat - Selects the drives only if the requested format is empty/satisfied already.
func FilterDrivesByRequestFormat(csiDrives []direct_csi.DirectCSIDrive) []direct_csi.DirectCSIDrive {
	filteredDriveList := []direct_csi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Spec.DirectCSIOwned && reflect.DeepEqual(csiDrive.Spec.RequestedFormat, direct_csi.RequestedFormat{}) {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByFsType - Filters the CSI drives by filesystem
func FilterDrivesByFsType(fsType string, csiDrives []direct_csi.DirectCSIDrive) []direct_csi.DirectCSIDrive {
	if fsType == "" {
		return csiDrives
	}
	filteredDriveList := []direct_csi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.Filesystem == fsType {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// SelectDriveByTopologyReq - selects the CSI drive by topology in the create volume request
func SelectDriveByTopologyReq(tReq *csi.TopologyRequirement, csiDrives []direct_csi.DirectCSIDrive) (direct_csi.DirectCSIDrive, error) {
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

	return direct_csi.DirectCSIDrive{}, status.Error(codes.ResourceExhausted, "Cannot satisfy the topology constraint")
}

func selectDriveByTopology(top *csi.Topology, csiDrives []direct_csi.DirectCSIDrive) (direct_csi.DirectCSIDrive, error) {
	topSegments := top.GetSegments()
	for _, csiDrive := range csiDrives {
		driveSegments := csiDrive.Status.Topology
		if matchSegments(topSegments, driveSegments) {
			return csiDrive, nil
		}
	}
	return direct_csi.DirectCSIDrive{}, status.Error(codes.ResourceExhausted, "Cannot satisfy the topology constraint")
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
