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

package installer

import (
	"context"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func removeVolumes(ctx context.Context, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := volume.ListVolumes(ctx, nil, nil, nil, nil, k8s.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = volume.ProcessVolumes(
		ctx,
		resultCh,
		func(volume *types.Volume) bool {
			return true
		},
		func(volume *types.Volume) error {
			volume.SetFinalizers([]string{})
			return nil
		},
		func(ctx context.Context, volume *types.Volume) error {
			if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
				return err
			}
			if err := client.VolumeClient().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		},
		nil,
		c.DryRun,
	)

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func removeDrives(ctx context.Context, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)

	defer cancelFunc()

	resultCh, err := drive.ListDrives(ctx, nil, nil, nil, nil, k8s.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = drive.ProcessDrives(
		ctx,
		resultCh,
		func(drive *types.Drive) bool {
			return true
		},
		func(drive *types.Drive) error {
			drive.SetFinalizers([]string{})
			return nil
		},
		func(ctx context.Context, drive *types.Drive) error {
			if _, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			if err := client.DriveClient().Delete(ctx, drive.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		},
		nil,
		c.DryRun,
	)

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func installCRDDefault(ctx context.Context, c *Config) error {
	return registerCRDs(ctx, c)
}

func uninstallCRDDefault(ctx context.Context, c *Config) error {
	if !c.UninstallCRD {
		return nil
	}

	if c.ForceRemove {
		if err := removeVolumes(ctx, c); err != nil {
			return err
		}
		if err := removeDrives(ctx, c); err != nil {
			return err
		}
	}

	return unregisterCRDs(ctx)
}
