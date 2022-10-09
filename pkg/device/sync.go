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

package device

import (
	"context"
	"errors"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type device struct {
	Device
	FSUUID        string
	Label         string
	TotalCapacity int64
	FreeCapacity  int64
}

func probeDeviceMap() (map[string][]device, error) {
	devices, err := Probe()
	if err != nil {
		return nil, err
	}

	deviceMap := map[string][]device{}
	for _, dev := range devices {
		if dev.Hidden || dev.Partitioned || len(dev.Holders) != 0 || dev.SwapOn || dev.CDROM {
			continue
		}

		fsuuid, label, totalCapacity, freeCapacity, err := xfs.Probe(utils.AddDevPrefix(dev.Name))
		if err != nil {
			if !errors.Is(err, xfs.ErrFSNotFound) {
				return nil, err
			}
			continue
		}

		deviceMap[fsuuid] = append(deviceMap[fsuuid], device{
			Device:        dev,
			FSUUID:        fsuuid,
			Label:         label,
			TotalCapacity: int64(totalCapacity),
			FreeCapacity:  int64(freeCapacity),
		})
	}

	return deviceMap, nil
}

func Sync(ctx context.Context) error {
	deviceMap, err := probeDeviceMap()
	if err != nil {
		return err
	}

	drives, err := drive.GetDriveList(ctx, nil, nil, nil, nil)
	if err != nil {
		return err
	}

	for i := range drives {
		var drive *types.Drive
		var mountCheck bool
		updateFunc := func() error {
			mountCheck = false

			if drive, err = client.DriveClient().Get(ctx, drives[i].Name, metav1.GetOptions{}); err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}

				return err
			}

			updated := true
			switch devices := deviceMap[drive.Status.FSUUID]; len(devices) {
			case 0:
				drive.Status.Status = directpvtypes.DriveStatusLost
			case 1:
				mountCheck = true
				changed := false
				if string(drive.GetDriveName()) != devices[0].Name {
					changed = true
					drive.SetDriveName(directpvtypes.DriveName(devices[0].Name))
				}
				switch {
				case drive.Status.TotalCapacity > devices[0].TotalCapacity:
					changed = true
					drive.Status.TotalCapacity = devices[0].TotalCapacity
					drive.Status.FreeCapacity = drive.Status.TotalCapacity - drive.Status.AllocatedCapacity
					if drive.Status.FreeCapacity < 0 {
						drive.Status.FreeCapacity = 0
					}
				case drive.Status.TotalCapacity < devices[0].TotalCapacity:
					changed = true
					drive.Status.TotalCapacity = devices[0].TotalCapacity
					drive.Status.FreeCapacity = drive.Status.TotalCapacity - drive.Status.AllocatedCapacity
				}
				if drive.Status.Make != devices[0].Make() {
					changed = true
					drive.Status.Make = devices[0].Make()
				}
				updated = changed
			default:
				drive.Status.Status = directpvtypes.DriveStatusError
				// FIXME: add/update conditions to denote too many devices are found by FSUUID
			}

			if updated {
				_, err = client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
			}

			return err
		}

		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return err
		}

		if mountCheck {
			if err = xfs.Mount(utils.AddDevPrefix(string(drive.GetDriveName())), types.GetVolumeRootDir(drive.Status.FSUUID)); err != nil {
				return err
			}
		}
	}

	return nil
}
