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
	"fmt"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/volume"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func installCRDDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, "CRD", registerCRDs); err != nil {
		return fmt.Errorf("unable to register CRD; %v", err)
	}
	return nil
}

func uninstallCRDDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, "CRD", deleteCRDDefault); err != nil {
		return fmt.Errorf("unable to delete CRD; %v", err)
	}
	return nil
}

func deleteCRDDefault(ctx context.Context, c *Config) error {
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

	return unregisterCRDs(ctx, c)
}

func removeVolumes(ctx context.Context, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range volume.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}

		result.Volume.RemovePVProtection()
		result.Volume.RemovePurgeProtection()

		if c.DryRun {
			continue
		}

		if _, err := client.VolumeClient().Update(ctx, &result.Volume, metav1.UpdateOptions{}); err != nil {
			return err
		}
		if err := client.VolumeClient().Delete(ctx, result.Volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func removeDrives(ctx context.Context, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for result := range drive.NewLister().List(ctx) {
		if result.Err != nil {
			if apierrors.IsNotFound(result.Err) {
				break
			}
			return result.Err
		}

		result.Drive.RemoveFinalizers()
		if c.DryRun {
			continue
		}

		if _, err := client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err := client.DriveClient().Delete(ctx, result.Drive.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
