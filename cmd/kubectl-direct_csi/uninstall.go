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
)

func init() {
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "register crds along with installation")
}

func uninstall(ctx context.Context, args []string) error {
	utils.Init()
	bold := color.New(color.Bold).SprintFunc()

	if uninstallCRD {
		if err := unregisterCRDs(ctx); err != nil {
			return err
		}
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
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' daemonset deleted", utils.Bold(identity))

	if err := utils.DeleteDeployment(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' deployment deleted", utils.Bold(identity))
	return nil
}
