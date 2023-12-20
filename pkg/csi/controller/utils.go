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
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func matchDrive(drive *types.Drive, req *csi.CreateVolumeRequest) bool {
	// Skip terminating drives
	if !drive.GetDeletionTimestamp().IsZero() {
		return false
	}

	// Skip drives if status is not ready
	if drive.Status.Status != directpvtypes.DriveStatusReady {
		return false
	}

	// Skip drives if unschedulable
	if drive.IsUnschedulable() {
		return false
	}

	// Match drive if it has requested capacity.
	if req.GetCapacityRange() != nil && drive.Status.FreeCapacity < req.GetCapacityRange().GetRequiredBytes() {
		return false
	}

	// Match drive by access-tier if requested.
	labels := drive.GetLabels()
	for key, value := range req.GetParameters() {
		// TODO: add migration doc to change "direct-csi-min-io/" prefixed access-tier parameters in custom storage classes
		if !strings.HasPrefix(key, consts.GroupName) {
			continue
		}
		switch key {
		case string(directpvtypes.AccessTierLabelKey):
			accessTiers, _ := directpvtypes.StringsToAccessTiers(value)
			if len(accessTiers) > 0 && drive.GetAccessTier() != accessTiers[0] {
				return false
			}
		case string(directpvtypes.VolumeClaimIDLabelKey):
			if drive.HasVolumeClaimID(value) {
				// Do not allocate another volume with this claim id
				return false
			}
		default:
			if labels[key] != value {
				return false
			}
		}
	}

	matchTopologies := func(topologies []*csi.Topology) bool {
		for _, topology := range topologies {
			for key, value := range topology.GetSegments() {
				if driveValue, found := drive.Status.Topology[key]; !found || value != driveValue {
					return false
				}
			}
		}
		return true
	}

	// Match drive by preferred topologies if requested.
	if len(req.GetAccessibilityRequirements().GetPreferred()) > 0 && matchTopologies(req.GetAccessibilityRequirements().GetPreferred()) {
		return true
	}

	// Match drive by requisite topology if requested.
	if len(req.GetAccessibilityRequirements().GetRequisite()) > 0 && matchTopologies(req.GetAccessibilityRequirements().GetRequisite()) {
		return true
	}

	// Match drive if no topology constraints requested.
	return len(req.GetAccessibilityRequirements().GetPreferred()) == 0 && len(req.GetAccessibilityRequirements().GetRequisite()) == 0
}

func getFilteredDrives(ctx context.Context, req *csi.CreateVolumeRequest) (drives []types.Drive, err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range client.NewDriveLister().List(ctx) {
		if result.Err != nil {
			return nil, result.Err
		}

		if result.Drive.VolumeExist(req.GetName()) {
			return []types.Drive{result.Drive}, nil
		}

		if matchDrive(&result.Drive, req) {
			drives = append(drives, result.Drive)
		}
	}

	return drives, nil
}

func selectDrive(ctx context.Context, req *csi.CreateVolumeRequest) (*types.Drive, error) {
	drives, err := getFilteredDrives(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(drives) == 0 {
		if len(req.GetAccessibilityRequirements().GetPreferred()) != 0 || len(req.GetAccessibilityRequirements().GetRequisite()) != 0 {
			requestedSize := "nil"
			if req.GetCapacityRange() != nil {
				requestedSize = fmt.Sprintf("%d bytes", req.GetCapacityRange().GetRequiredBytes())
			}
			var requestedNodes []string
			if requestedNodes = getNodeNamesFromTopology(req.AccessibilityRequirements.GetPreferred()); len(requestedNodes) == 0 {
				requestedNodes = getNodeNamesFromTopology(req.AccessibilityRequirements.GetRequisite())
			}
			return nil, status.Errorf(codes.ResourceExhausted, "no drive found for requested topology; requested node(s): %s; requested size: %s", strings.Join(requestedNodes, ","), requestedSize)
		}
		if req.GetCapacityRange() != nil {
			return nil, status.Errorf(codes.OutOfRange, "no drive found for requested size %v", req.GetCapacityRange().GetRequiredBytes())
		}
		return nil, status.Error(codes.FailedPrecondition, "no drive found")
	}

	maxFreeCapacity := int64(-1)
	var maxFreeCapacityDrives []types.Drive
	for _, drive := range drives {
		switch {
		case drive.Status.FreeCapacity == maxFreeCapacity:
			maxFreeCapacityDrives = append(maxFreeCapacityDrives, drive)
		case drive.Status.FreeCapacity > maxFreeCapacity:
			maxFreeCapacity = drive.Status.FreeCapacity
			maxFreeCapacityDrives = []types.Drive{drive}
		}
	}

	if len(maxFreeCapacityDrives) == 1 {
		return &maxFreeCapacityDrives[0], nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(maxFreeCapacityDrives))))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "random number generation failed; %v", err)
	}

	return &maxFreeCapacityDrives[n.Int64()], nil
}

func getNodeNamesFromTopology(topologies []*csi.Topology) (requestedNodes []string) {
	for _, topology := range topologies {
		for key, value := range topology.GetSegments() {
			if key == string(directpvtypes.TopologyDriverNode) {
				requestedNodes = append(requestedNodes, value)
				break
			}
		}
	}
	return
}
