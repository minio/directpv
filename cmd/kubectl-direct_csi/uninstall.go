// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/minio/direct-csi/pkg/utils"
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
	dangerous    = false
	unfinalize   = false
)

func init() {
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "unregister direct.csi.min.io group crds")

	uninstallCmd.PersistentFlags().BoolVarP(&dangerous, "dangerous", "", dangerous, "potentially dangerous operation. May cause data loss. Required for --unfinalize to work")
	uninstallCmd.PersistentFlags().BoolVarP(&unfinalize, "unfinalize", "", unfinalize, "remove all finalizers from direct.csi.min.io resources. only works along with --dangerous")

	uninstallCmd.PersistentFlags().MarkHidden("dangerous")
	uninstallCmd.PersistentFlags().MarkHidden("unfinalize")

}

func uninstall(ctx context.Context, args []string) error {
	utils.Init()
	bold := color.New(color.Bold).SprintFunc()
	directCSIClient := utils.GetDirectCSIClient()

	if uninstallCRD {
		if unfinalize && dangerous {
			volumes, err := directCSIClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			for _, v := range volumes.Items {
				v.ObjectMeta.SetFinalizers([]string{})
				if _, err := directCSIClient.DirectCSIVolumes().Update(ctx, &v, metav1.UpdateOptions{}); err != nil {
					return err
				}
				if err := directCSIClient.DirectCSIVolumes().Delete(ctx, v.Name, metav1.DeleteOptions{}); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
				}
			}
			drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}
			for _, d := range drives.Items {
				d.ObjectMeta.SetFinalizers([]string{})
				if _, err := directCSIClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
					return err
				}
				if err := directCSIClient.DirectCSIDrives().Delete(ctx, d.Name, metav1.DeleteOptions{}); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
				}
			}
		}
		if err := unregisterCRDs(ctx); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
		glog.Infof("'%s' crds deleted", bold(identity))
	}

	if err := utils.DeleteNamespace(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' namespace deleted", bold(identity))

	if err := utils.DeleteCSIDriver(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' csidriver deleted", bold(identity))

	if err := utils.DeleteStorageClass(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' storageclass deleted", bold(identity))

	if err := utils.DeleteService(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' service deleted", bold(identity))

	if err := utils.RemoveRBACRoles(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' rbac roles deleted", utils.Bold(identity))

	if err := utils.DeleteDaemonSet(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' daemonset deleted", utils.Bold(identity))

	if err := utils.DeleteDeployment(ctx, identity); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	glog.Infof("'%s' deployment deleted", utils.Bold(identity))
	return nil
}
