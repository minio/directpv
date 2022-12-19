// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package node

import (
	"context"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func probeDevices(nodeID directpvtypes.NodeID) ([]types.Device, error) {
	devices, err := device.Probe()
	if err != nil {
		return nil, err
	}
	var nodeDevices []types.Device
	for i := range devices {
		nodeDevices = append(nodeDevices, devices[i].ToNodeDevice(nodeID))
	}
	return nodeDevices, nil
}

// Sync probes the local devices and syncs the DirectPVNode CRD objects with the probed information.
func Sync(ctx context.Context, nodeID directpvtypes.NodeID) error {
	devices, err := probeDevices(nodeID)
	if err != nil {
		return err
	}
	updateFunc := func() error {
		nodeClient := client.NodeClient()
		node, err := nodeClient.Get(ctx, string(nodeID), metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
			_, err = nodeClient.Create(ctx, types.NewNode(nodeID, devices), metav1.CreateOptions{})
			return err
		}
		node.Status.Devices = devices
		node.Spec.Refresh = false
		if _, err := nodeClient.Update(ctx, node, metav1.UpdateOptions{TypeMeta: types.NewNodeTypeMeta()}); err != nil {
			return err
		}
		return nil
	}
	if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
		return err
	}
	return nil
}
