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
	"math/big"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/matcher"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func matchDrive(drive directcsi.DirectCSIDrive, req *csi.CreateVolumeRequest) bool {
	// Match drive only if it is Ready
	if utils.IsConditionStatus(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionReady),
		metav1.ConditionFalse) {
		return false
	}

	// skip terminating drives
	if !drive.GetDeletionTimestamp().IsZero() {
		return false
	}

	// Match drive only in Ready or InUse state.
	switch drive.Status.DriveStatus {
	case directcsi.DriveStatusReady, directcsi.DriveStatusInUse:
	default:
		return false
	}

	// Match drive if it has requested capacity.
	if req.GetCapacityRange() != nil && drive.Status.FreeCapacity < req.GetCapacityRange().GetRequiredBytes() {
		return false
	}

	// Match drive if requested filesystem matches.
	if len(req.GetVolumeCapabilities()) > 0 && drive.Status.Filesystem != req.GetVolumeCapabilities()[0].GetMount().GetFsType() {
		return false
	}

	// Match drive by access-tier if requested.
	for key, value := range req.GetParameters() {
		if key == "direct-csi-min-io/access-tier" && string(drive.Status.AccessTier) != value {
			return false
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

func getFilteredDrives(ctx context.Context, req *csi.CreateVolumeRequest) (drives []directcsi.DirectCSIDrive, err error) {
	resultCh, err := client.ListDrives(ctx, nil, nil, nil, client.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	for result := range resultCh {
		if result.Err != nil {
			return nil, result.Err
		}

		if matcher.StringIn(result.Drive.Finalizers, directcsi.DirectCSIDriveFinalizerPrefix+req.GetName()) {
			return []directcsi.DirectCSIDrive{result.Drive}, nil
		}

		if matchDrive(result.Drive, req) {
			drives = append(drives, result.Drive)
		}
	}

	return drives, nil
}

func selectDrive(ctx context.Context, req *csi.CreateVolumeRequest) (*directcsi.DirectCSIDrive, error) {
	drives, err := getFilteredDrives(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(drives) == 0 {
		if len(req.GetAccessibilityRequirements().GetPreferred()) != 0 || len(req.GetAccessibilityRequirements().GetRequisite()) != 0 {
			return nil, status.Error(codes.ResourceExhausted, "no drive found for requested topology")
		}

		if req.GetCapacityRange() != nil {
			return nil, status.Errorf(codes.OutOfRange, "no drive found for requested size %v", req.GetCapacityRange().GetRequiredBytes())
		}

		return nil, status.Error(codes.FailedPrecondition, "no drive found")
	}

	maxFreeCapacity := int64(-1)
	var maxFreeCapacityDrives []directcsi.DirectCSIDrive
	for _, drive := range drives {
		switch {
		case drive.Status.FreeCapacity == maxFreeCapacity:
			maxFreeCapacityDrives = append(maxFreeCapacityDrives, drive)
		case drive.Status.FreeCapacity > maxFreeCapacity:
			maxFreeCapacity = drive.Status.FreeCapacity
			maxFreeCapacityDrives = []directcsi.DirectCSIDrive{drive}
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
