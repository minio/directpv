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
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
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
		return []directcsi.DirectCSIDrive{}, status.Error(codes.FailedPrecondition, "No drives are 'Ready' to be used. Please use `kubectl direct-csi drives format` command to format the drives")
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
			accessT, err := directcsi.ToAccessTier(v)
			if err != nil {
				return csiDrives, err
			}
			filteredDriveList = filterDrivesByAccessTier(accessT, filteredDriveList)
		default:
		}
	}
	return filteredDriveList, nil
}

func filterDrivesByAccessTier(accessTier directcsi.AccessTier, csiDrives []directcsi.DirectCSIDrive) []directcsi.DirectCSIDrive {
	filteredDriveList := []directcsi.DirectCSIDrive{}
	for _, csiDrive := range csiDrives {
		if csiDrive.Status.AccessTier == accessTier {
			filteredDriveList = append(filteredDriveList, csiDrive)
		}
	}
	return filteredDriveList
}

// FilterDrivesByTopologyRequirements - selects the CSI drive by topology in the create volume request
func FilterDrivesByTopologyRequirements(volReq *csi.CreateVolumeRequest, csiDrives []directcsi.DirectCSIDrive, nodeID string) (directcsi.DirectCSIDrive, error) {
	tReq := volReq.GetAccessibilityRequirements()

	preferredXs := tReq.GetPreferred()
	requisiteXs := tReq.GetRequisite()

	// Try to fulfill the preferred topology request, If not, fallback to requisite list.
	// Ref: https://godoc.org/github.com/container-storage-interface/spec/lib/go/csi#TopologyRequirement
	for _, preferredTop := range preferredXs {
		if selectedDrives, err := selectDrivesByTopology(preferredTop, csiDrives); err == nil {
			return selectDriveByFreeCapacity(selectedDrives)
		}
	}

	for _, requisiteTop := range requisiteXs {
		if selectedDrives, err := selectDrivesByTopology(requisiteTop, csiDrives); err == nil {
			return selectDriveByFreeCapacity(selectedDrives)
		}
	}

	if len(preferredXs) == 0 && len(requisiteXs) == 0 {
		return selectDriveByFreeCapacity(csiDrives)
	}

	klog.V(3).InfoS("Cannot satisfy the topology constraint",
		"volume", volReq.GetName(),
		"preferredTopology", preferredXs,
		"requisiteTopology", requisiteXs,
	)

	message := fmt.Sprintf("No suitable drive found on node %v for %v. ", nodeID, volReq.GetName()) +
		"Use nodeSelector or affinity to restrict pods to run on node with enough capacity"
	return directcsi.DirectCSIDrive{}, status.Error(codes.ResourceExhausted, message)
}

func selectDriveByFreeCapacity(csiDrives []directcsi.DirectCSIDrive) (directcsi.DirectCSIDrive, error) {
	// Sort the drives by free capacity [Descending]
	sort.SliceStable(csiDrives, func(i, j int) bool {
		return csiDrives[i].Status.FreeCapacity > csiDrives[j].Status.FreeCapacity
	})

	groupByFreeCapacity := func() []directcsi.DirectCSIDrive {
		maxFreeCapacity := csiDrives[0].Status.FreeCapacity
		groupedDrives := []directcsi.DirectCSIDrive{}
		for _, csiDrive := range csiDrives {
			if csiDrive.Status.FreeCapacity == maxFreeCapacity {
				groupedDrives = append(groupedDrives, csiDrive)
			}
		}
		return groupedDrives
	}

	pickRandomIndex := func(max int) (int, error) {
		rInt, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
		if err != nil {
			return int(0), err
		}
		return int(rInt.Int64()), nil
	}

	selectedDrives := groupByFreeCapacity()
	rIndex, err := pickRandomIndex(len(selectedDrives))
	if err != nil {
		return selectedDrives[rIndex], status.Errorf(codes.Internal, "Error while selecting (random) drive: %v", err)
	}
	return selectedDrives[rIndex], nil
}

func selectDrivesByTopology(top *csi.Topology, csiDrives []directcsi.DirectCSIDrive) ([]directcsi.DirectCSIDrive, error) {
	matchingDriveList := []directcsi.DirectCSIDrive{}
	topSegments := top.GetSegments()
	for _, csiDrive := range csiDrives {
		driveSegments := csiDrive.Status.Topology
		if matchSegments(topSegments, driveSegments) {
			matchingDriveList = append(matchingDriveList, csiDrive)
		}
	}
	return matchingDriveList, func() error {
		if len(matchingDriveList) == 0 {
			return status.Error(codes.ResourceExhausted, "Cannot satisfy the topology constraint")
		}
		return nil
	}()
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
