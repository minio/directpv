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

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/minio/direct-csi/pkg/installer"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/apimachinery/pkg/api/errors"
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

func init() {
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "unregister direct.csi.min.io group crds")
	uninstallCmd.PersistentFlags().BoolVarP(&forceRemove, "force", "", forceRemove, "Removes the direct.csi.min.io resources [May cause data loss]")
}

func uninstall(ctx context.Context, args []string) error {
	if dryRun {
		klog.Errorf("'--dry-run' flag is not supported for uninstall")
		return nil
	}

	bold := color.New(color.Bold).SprintFunc()
	directCSIClient := utils.GetDirectCSIClient()

	if uninstallCRD {
		volumes, err := utils.GetVolumeList(ctx, directCSIClient.DirectCSIVolumes(), nil, nil, nil, nil)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}

		if len(volumes) > 0 && !forceRemove {
			klog.Errorf("Cannot unregister CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
			return nil
		}

		for _, v := range volumes {
			v.SetFinalizers([]string{})
			if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, &v, metav1.UpdateOptions{}); err != nil {
				return err
			}
			if err := directCSIClient.DirectCSIVolumes().Delete(ctx, v.Name, metav1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}
		drives, err := utils.GetDriveList(ctx, directCSIClient.DirectCSIDrives(), nil, nil, nil)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}

		if len(drives) > 0 && !forceRemove {
			klog.Errorf("Cannot unregister CRDs. Please use `%s` to delete the resources", utils.Bold("--force"))
			return nil
		}

		for _, d := range drives {
			d.SetFinalizers([]string{})
			if _, err := directCSIClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
				return err
			}
			if err := directCSIClient.DirectCSIDrives().Delete(ctx, d.Name, metav1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}
		if forceRemove {
			klog.Infof("'%s' CRD resources deleted", bold(identity))
		}

		if err := unregisterCRDs(ctx); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		klog.Infof("'%s' crds deleted", bold(identity))
	}

	if err := installer.DeleteCSIDriver(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' csidriver deleted", bold(identity))

	if err := installer.DeleteStorageClass(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' storageclass deleted", bold(identity))

	if err := installer.DeleteService(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' service deleted", bold(identity))

	if err := installer.RemoveRBACRoles(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' rbac roles deleted", utils.Bold(identity))

	if err := installer.DeleteDaemonSet(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' daemonset deleted", utils.Bold(identity))

	if err := installer.DeleteDriveValidationRules(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := installer.DeleteControllerSecret(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' drive validation rules removed", utils.Bold(identity))

	if err := installer.DeleteControllerDeployment(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' controller deployment deleted", utils.Bold(identity))

	if err := installer.DeleteConversionDeployment(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := installer.DeleteConversionSecret(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := installer.DeleteConversionWebhookCertsSecret(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	klog.Infof("'%s' conversion deployment deleted", utils.Bold(identity))

	if err := installer.DeleteNamespace(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	klog.Infof("'%s' namespace deleted", bold(identity))

	return nil
}
