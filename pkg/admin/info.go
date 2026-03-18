// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/directpv/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeLevelInfo represents the node level information
type NodeLevelInfo struct {
	DriveSize   uint64
	VolumeSize  uint64
	DriveCount  int
	VolumeCount int
}

// Info returns the overall info of the directpv installation
func (client *Client) Info(ctx context.Context) (map[string]NodeLevelInfo, error) {
	crds, err := client.CRD().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list CRDs; %w", err)
	}
	drivesFound := false
	volumesFound := false
	for _, crd := range crds.Items {
		if strings.Contains(crd.Name, consts.DriveResource+"."+consts.GroupName) {
			drivesFound = true
		}
		if strings.Contains(crd.Name, consts.VolumeResource+"."+consts.GroupName) {
			volumesFound = true
		}
	}
	if !drivesFound || !volumesFound {
		return nil, fmt.Errorf("%v installation not found", consts.AppPrettyName)
	}
	nodeList, err := client.K8s().GetCSINodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get CSI nodes; %w", err)
	}
	if len(nodeList) == 0 {
		return nil, fmt.Errorf("%v not installed", consts.AppPrettyName)
	}
	drives, err := client.NewDriveLister().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get drive list; %w", err)
	}
	volumes, err := client.NewVolumeLister().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get volume list; %w", err)
	}
	nodeInfo := make(map[string]NodeLevelInfo, len(nodeList))
	for _, n := range nodeList {
		driveCount := 0
		driveSize := uint64(0)
		for _, d := range drives {
			if string(d.GetNodeID()) == n {
				driveCount++
				driveSize += uint64(d.Status.TotalCapacity)
			}
		}
		volumeCount := 0
		volumeSize := uint64(0)
		for _, v := range volumes {
			if string(v.GetNodeID()) == n {
				if v.IsPublished() {
					volumeCount++
					volumeSize += uint64(v.Status.TotalCapacity)
				}
			}
		}
		nodeInfo[n] = NodeLevelInfo{
			DriveSize:   driveSize,
			VolumeSize:  volumeSize,
			DriveCount:  driveCount,
			VolumeCount: volumeCount,
		}
	}
	return nodeInfo, nil
}
