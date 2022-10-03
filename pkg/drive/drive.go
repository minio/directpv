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

package drive

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDrive(name string, status types.DriveStatus) *types.Drive {
	drive := &types.Drive{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Finalizers: []string{consts.DriveFinalizerDataProtection},
		},
		Status: status,
	}

	types.UpdateLabels(drive, map[types.LabelKey]types.LabelValue{
		types.NodeLabelKey:       types.NewLabelValue(status.NodeName),
		types.PathLabelKey:       types.NewLabelValue(utils.TrimDevPrefix(status.Path)),
		types.VersionLabelKey:    types.NewLabelValue(consts.LatestAPIVersion),
		types.CreatedByLabelKey:  consts.DriverName,
		types.AccessTierLabelKey: types.NewLabelValue(string(status.AccessTier)),
	})

	return drive
}

// CreateDrive creates drive CRD.
func CreateDrive(ctx context.Context, drive *types.Drive) error {
	_, err := client.DriveClient().Create(ctx, drive, metav1.CreateOptions{})
	return err
}

// DeleteDrive deletes drive CRD.
func DeleteDrive(ctx context.Context, drive *types.Drive, force bool) error {
	finalizers := drive.GetFinalizers()
	switch len(finalizers) {
	case 1:
		if finalizers[0] != consts.DriveFinalizerDataProtection {
			return fmt.Errorf("invalid state reached. Report this issue at https://github.com/minio/directpv/issues")
		}

		if err := sys.Unmount(types.GetDriveMountDir(drive.Status.FSUUID), false, false, false); err != nil {
			return err
		}

		drive.Finalizers = []string{}
		_, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
		return err
	case 0:
		return nil
	default:
		for _, finalizer := range finalizers {
			if !strings.HasPrefix(finalizer, consts.DriveFinalizerPrefix) {
				continue
			}
			volumeName := strings.TrimPrefix(finalizer, consts.DriveFinalizerPrefix)
			volume, err := client.VolumeClient().Get(
				ctx, volumeName, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}

			volume.Status.SetDriveLost()
			_, err = client.VolumeClient().Update(
				ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
			)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
