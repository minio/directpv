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

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/utils/installer"
)

var installCmd = &cobra.Command{
	Use:          "install",
	Short:        "Install direct-csi in k8s cluster",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return install(c.Context(), args)
	},
}

var (
	installCRD   = false
	overwriteCRD = false
	image        = "minio/direct-csi:" + Version
)

func init() {
	installCmd.PersistentFlags().BoolVarP(&installCRD, "crd", "c", installCRD, "register crds along with installation")
	installCmd.PersistentFlags().BoolVarP(&overwriteCRD, "force", "f", overwriteCRD, "delete and recreate CRDs")
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, "direct-csi image")
}

func install(ctx context.Context, args []string) error {
	utils.Init()

	if installCRD {
	crdInstall:
		if err := registerCRDs(ctx); err != nil {
			if !errors.IsAlreadyExists(err) {
				return err
			}
			// if it exists
			if overwriteCRD {
				if err := unregisterCRDs(ctx); err != nil {
					if !errors.IsNotFound(err) {
						return err
					}
					goto crdInstall
				}
			}
		}
		glog.Infof("crds successfully registered")
	}

	if err := installer.CreateNamespace(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' namespace created", utils.Bold(identity))

	if err := installer.CreateCSIDriver(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' csidriver created", utils.Bold(identity))

	if err := installer.CreateStorageClass(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' storageclass created", utils.Bold(identity))

	if err := installer.CreateService(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' service created", utils.Bold(identity))

	if err := installer.CreateRBACRoles(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' rbac roles created", utils.Bold(identity))

	if err := installer.CreateDaemonSet(ctx, identity, image); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' daemonset created", utils.Bold(identity))

	if err := installer.CreateDeployment(ctx, identity, image); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' deployment created", utils.Bold(identity))

	if err := installer.RegisterDriveValidationRules(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' drive validation rules registered", utils.Bold(identity))

	return nil
}
