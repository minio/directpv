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
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const minSupportedDeviceSize = 512 * 1024 * 1024 // 512 MiB

func newDevice(dev device.Device, nodeID directpvtypes.NodeID) types.Device {
	var reasons []string

	if dev.Size < minSupportedDeviceSize {
		reasons = append(reasons, "Too small")
	}

	if dev.Hidden {
		reasons = append(reasons, "Hidden")
	}

	if dev.ReadOnly {
		reasons = append(reasons, "Read only")
	}

	if dev.Partitioned {
		reasons = append(reasons, "Partitioned")
	}

	if len(dev.Holders) != 0 {
		reasons = append(reasons, "Held by other device")
	}

	if len(dev.MountPoints) != 0 {
		reasons = append(reasons, "Mounted")
	}

	if dev.SwapOn {
		reasons = append(reasons, "Swap")
	}

	if dev.CDROM {
		reasons = append(reasons, "CDROM")
	}

	if dev.UDevData["ID_FS_TYPE"] == "xfs" && dev.UDevData["ID_FS_UUID"] != "" {
		if _, err := client.DriveClient().Get(context.Background(), dev.UDevData["ID_FS_UUID"], metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				reasons = append(reasons, "internal error; "+err.Error())
			}
		} else {
			reasons = append(reasons, "Used by "+consts.AppPrettyName)
		}
	}

	var deniedReason string
	if len(reasons) != 0 {
		deniedReason = strings.Join(reasons, "; ")
	}
	return types.Device{
		Name:         dev.Name,
		ID:           dev.ID(nodeID),
		MajorMinor:   dev.MajorMinor,
		Size:         dev.Size,
		Make:         dev.Make(),
		FSType:       dev.FSType(),
		FSUUID:       dev.FSUUID(),
		DeniedReason: deniedReason,
	}
}

func probeDevices(nodeID directpvtypes.NodeID) ([]types.Device, error) {
	devices, err := device.Probe()
	if err != nil {
		return nil, err
	}
	var nodeDevices []types.Device
	for i := range devices {
		nodeDevices = append(nodeDevices, newDevice(devices[i], nodeID))
	}
	return nodeDevices, nil
}

// Sync - syncs the node with locally probed devices
func Sync(ctx context.Context, nodeID directpvtypes.NodeID, retryOnConfict bool) error {
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
	if !retryOnConfict {
		return updateFunc()
	}
	if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
		return err
	}
	return nil
}
