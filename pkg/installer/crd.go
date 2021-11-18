// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"errors"
	"fmt"
	"os"
	"time"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/client"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var errForceRequired = errors.New("force option required to remove the CRD resources")

func removeVolumes(ctx context.Context, directCSIClient clientset.DirectV1beta3Interface, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := client.ListVolumes(ctx, directCSIClient.DirectCSIVolumes(), nil, nil, nil, nil, client.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	defaultAuditDir, err := utils.GetDefaultAuditDir()
	if err != nil {
		return fmt.Errorf("unable to get default audit directory; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0700); err != nil {
		return err
	}
	file, err := utils.NewSafeFile(fmt.Sprintf("%v/%v-%v", defaultAuditDir, "removeVolumes", time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("unable to get default audit directory ; %w", err)
	}

	defer func() {
		if cerr := file.Close(); err != nil {
			klog.Errorf("unable to close file; %w", cerr)
		} else {
			err = cerr
		}
	}()

	err = client.ProcessVolumes(
		ctx,
		resultCh,
		func(volume *directcsi.DirectCSIVolume) bool {
			return true
		},
		func(volume *directcsi.DirectCSIVolume) error {
			if !c.ForceRemove {
				return errForceRequired
			}
			volume.SetFinalizers([]string{})
			return nil
		},
		func(ctx context.Context, volume *directcsi.DirectCSIVolume) error {
			if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
				return err
			}
			if err := directCSIClient.DirectCSIVolumes().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		},
		file,
		c.DryRun,
	)

	if errors.Is(err, errForceRequired) {
		klog.Errorf("Cannot unregister DirectCSIVolume CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
	}

	return err
}

func removeDrives(ctx context.Context, directCSIClient clientset.DirectV1beta3Interface, c *Config) error {
	ctx, cancelFunc := context.WithCancel(ctx)

	defer cancelFunc()

	resultCh, err := client.ListDrives(ctx, directCSIClient.DirectCSIDrives(), nil, nil, nil, client.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	defaultAuditDir, err := utils.GetDefaultAuditDir()
	if err != nil {
		return fmt.Errorf("unable to get default audit directory; %w", err)
	}
	if err := os.MkdirAll(defaultAuditDir, 0700); err != nil {
		return err
	}
	file, err := utils.NewSafeFile(fmt.Sprintf("%v/%v-%v", defaultAuditDir, "removeDrives", time.Now().UnixNano()))
	if err != nil {
		return fmt.Errorf("unable to get default audit directory ; %w", err)
	}

	defer func() {
		if cerr := file.Close(); err != nil {
			klog.Errorf("unable to close file; %w", cerr)
		} else {
			err = cerr
		}
	}()

	err = client.ProcessDrives(
		ctx,
		resultCh,
		func(drive *directcsi.DirectCSIDrive) bool {
			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			if !c.ForceRemove {
				return errForceRequired
			}
			drive.SetFinalizers([]string{})
			return nil
		},
		func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
			if _, err := directCSIClient.DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			if err := directCSIClient.DirectCSIDrives().Delete(ctx, drive.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		},
		file,
		c.DryRun,
	)

	if errors.Is(err, errForceRequired) {
		klog.Errorf("Cannot unregister DirectCSIDrive CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
	}

	return err
}

func installCRDDefault(ctx context.Context, c *Config) error {
	if err := registerCRDs(ctx, c); err != nil {
		return err
	}

	if !c.DryRun {
		klog.Infof("crds successfully registered")
	}

	return nil
}

func uninstallCRDDefault(ctx context.Context, c *Config) error {
	if !c.UninstallCRD {
		return nil
	}
	directCSIClient := client.GetDirectCSIClient()
	if err := removeVolumes(ctx, directCSIClient, c); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := removeDrives(ctx, directCSIClient, c); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if c.ForceRemove {
		klog.Infof("'%s' CRD resources deleted", utils.Bold(c.Identity))
	}

	klog.Infof("'%s' CRDs deleted", utils.Bold(c.Identity))
	return unregisterCRDs(ctx)
}
