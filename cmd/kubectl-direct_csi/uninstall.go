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

package main

import (
	"context"
	"errors"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	clientset "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/installer"
	"github.com/minio/direct-csi/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
)

var uninstallCmd = &cobra.Command{
	Use:          "uninstall",
	Short:        "Uninstall direct-csi in k8s cluster",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return uninstall(c.Context(), args)
	},
}

var (
	uninstallCRD = false
	forceRemove  = false
)

var errForceRequired = errors.New("force option required")

func init() {
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "unregister direct.csi.min.io group crds")
	uninstallCmd.PersistentFlags().BoolVarP(&forceRemove, "force", "", forceRemove, "Removes the direct.csi.min.io resources [May cause data loss]")
}

func removeVolumes(ctx context.Context, directCSIClient clientset.DirectV1beta3Interface) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := utils.ListVolumes(ctx, directCSIClient.DirectCSIVolumes(), nil, nil, nil, nil, utils.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = processVolumes(
		ctx,
		resultCh,
		func(volume *directcsi.DirectCSIVolume) bool {
			return true
		},
		func(volume *directcsi.DirectCSIVolume) error {
			if !forceRemove {
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
	)

	if errors.Is(err, errForceRequired) {
		klog.Errorf("Cannot unregister DirectCSIVolume CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
		return nil
	}

	return err
}

func removeDrives(ctx context.Context, directCSIClient clientset.DirectV1beta3Interface) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := utils.ListDrives(ctx, directCSIClient.DirectCSIDrives(), nil, nil, nil, utils.MaxThreadCount)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = processDrives(
		ctx,
		resultCh,
		func(drive *directcsi.DirectCSIDrive) bool {
			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			if !forceRemove {
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
	)

	if errors.Is(err, errForceRequired) {
		klog.Errorf("Cannot unregister DirectCSIDrive CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
		return nil
	}

	return err
}

func uninstall(ctx context.Context, args []string) error {
	if dryRun {
		klog.Errorf("'--dry-run' flag is not supported for uninstall")
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	directCSIClient := utils.GetDirectCSIClient()

	if uninstallCRD {
		if err := removeVolumes(ctx, directCSIClient); err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if err := removeDrives(ctx, directCSIClient); err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if forceRemove {
			klog.Infof("'%s' CRD resources deleted", bold(identity))
		}

		if err := unregisterCRDs(ctx); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		klog.Infof("'%s' crds deleted", bold(identity))
	}

	if err := installer.DeleteCSIDriver(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' csidriver deleted", bold(identity))

	if err := installer.DeleteStorageClass(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' storageclass deleted", bold(identity))

	if err := installer.DeleteService(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' service deleted", bold(identity))

	if err := installer.RemoveRBACRoles(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' rbac roles deleted", utils.Bold(identity))

	if err := installer.DeleteDaemonSet(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' daemonset deleted", utils.Bold(identity))

	if err := installer.DeleteDriveValidationRules(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := installer.DeleteControllerSecret(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' drive validation rules removed", utils.Bold(identity))

	if err := installer.DeleteControllerDeployment(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' controller deployment deleted", utils.Bold(identity))

	if err := installer.DeleteLegacyConversionDeployment(ctx, identity); err != nil {
		return err
	}

	if err := installer.DeleteConversionSecrets(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' conversion secrets deleted", utils.Bold(identity))

	if err := installer.DeletePodSecurityPolicy(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' pod security policy removed", utils.Bold(identity))

	if err := installer.DeleteNamespace(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' namespace deleted", bold(identity))

	return nil
}
