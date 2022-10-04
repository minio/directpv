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

package volume

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func Stage(ctx context.Context, volume *types.Volume) error {
	if err := os.Mkdir(volume.Status.DataPath, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		// FIXME: handle I/O error and mark associated drive's status as ERROR.
		klog.ErrorS(err, "unable to create data path", "dataPath", volume.Status.DataPath)
		return err
	}

	if err := xfs.BindMount(volume.Status.DataPath, volume.Status.StagingTargetPath, false); err != nil {
		return fmt.Errorf("unable to bind mount volume directory to staging target path; %v", err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(volume.Status.TotalCapacity),
		SoftLimit: uint64(volume.Status.TotalCapacity),
	}

	device, err := device.GetDeviceByFSUUID(volume.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				"on the host to reload",
			"FSUUID", volume.Status.FSUUID)
		client.Eventf(
			volume, corev1.EventTypeWarning, "NodeStageVolume",
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", volume.Status.FSUUID)
		return fmt.Errorf("unable to find device by FSUUID %v; %v", volume.Status.FSUUID, err)
	}

	if err := xfs.SetQuota(ctx, device, volume.Status.StagingTargetPath, volume.Name, quota); err != nil {
		klog.ErrorS(err, "unable to set quota on staging target path", "StagingTargetPath", volume.Status.StagingTargetPath)
		return fmt.Errorf("unable to set quota on staging target path; %v", err)
	}
	return nil
}
