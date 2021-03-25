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
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	installCRD       = false
	overwriteCRD     = false
	admissionControl = false
	image            = "direct-csi:" + Version
	registry         = "quay.io"
	org              = "minio"
)

func init() {
	installCmd.PersistentFlags().BoolVarP(&installCRD, "crd", "c", installCRD, "register crds along with installation")
	installCmd.PersistentFlags().BoolVarP(&overwriteCRD, "force", "f", overwriteCRD, "delete and recreate CRDs")
	installCmd.PersistentFlags().StringVarP(&image, "image", "i", image, "direct-csi image")
	installCmd.PersistentFlags().StringVarP(&registry, "registry", "r", registry, "registry where direct-csi images are available")
	installCmd.PersistentFlags().StringVarP(&org, "org", "g", org, "organization name where direct-csi images are available")
	installCmd.PersistentFlags().BoolVarP(&admissionControl, "admission-control", "", admissionControl, "turn on direct-csi admission controller")

	installCmd.PersistentFlags().MarkDeprecated("crd", "Will be removed in version 1.5 or greater")
}

func install(ctx context.Context, args []string) error {
	if err := validImage(image); err != nil {
		return fmt.Errorf("invalid argument. format of '--image' must be [image:tag] err=%v", err)
	}
	if err := validOrg(org); err != nil {
		return fmt.Errorf("invalid org. format of '--org' must be [a-zA-Z][a-zA-Z0-9-.]* err=%v", err)
	}
	if err := validRegistry(registry); err != nil {
		return fmt.Errorf("invalid registry. format of '--registry' must be [host:port?]")
	}

	dryRun := viper.GetBool(dryRunFlagName)
	if !dryRun {
		utils.Init()
	}

	if err := installer.CreateNamespace(ctx, identity, dryRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' namespace created", utils.Bold(identity))
	}

	if err := installer.CreateConversionDeployment(ctx, identity, image, dryRun, registry, org); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' conversion deployment created", utils.Bold(identity))
	}

crdInstall:
	if err := registerCRDs(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		// if it exists
		if !dryRun && overwriteCRD {
			glog.Infof("overwriting CRDs")
			if err := unregisterCRDs(ctx); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
			glog.Infof("Writing CRDs again")
			goto crdInstall
		}
	}
	if !dryRun {
		glog.Infof("crds successfully registered")
	}

	if err := installer.CreateCSIDriver(ctx, identity, dryRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' csidriver created", utils.Bold(identity))
	}

	if err := installer.CreateStorageClass(ctx, identity, dryRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' storageclass created", utils.Bold(identity))
	}

	if err := installer.CreateService(ctx, identity, dryRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' service created", utils.Bold(identity))
	}

	if err := installer.CreateRBACRoles(ctx, identity, dryRun); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' rbac roles created", utils.Bold(identity))
	}

	if err := installer.CreateDaemonSet(ctx, identity, image, dryRun, registry, org); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' daemonset created", utils.Bold(identity))
	}

	if err := installer.CreateDeployment(ctx, identity, image, dryRun, registry, org); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	if !dryRun {
		glog.Infof("'%s' deployment created", utils.Bold(identity))
	}

	if admissionControl {
		if err := installer.RegisterDriveValidationRules(ctx, identity, dryRun); err != nil {
			if !errors.IsAlreadyExists(err) {
				return err
			}
		}
		if !dryRun {
			glog.Infof("'%s' drive validation rules registered", utils.Bold(identity))
		}
	}

	return nil
}
