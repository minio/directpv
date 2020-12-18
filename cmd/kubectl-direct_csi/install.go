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
)

func init() {
	installCmd.PersistentFlags().BoolVarP(&installCRD, "crd", "c", installCRD, "register crds along with installation")
	installCmd.PersistentFlags().BoolVarP(&overwriteCRD, "force", "f", overwriteCRD, "delete and recreate CRDs")
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

	if err := utils.CreateNamespace(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' namespace created", utils.Bold(identity))

	if err := utils.CreateCSIDriver(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' csidriver created", utils.Bold(identity))

	if err := utils.CreateStorageClass(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' storageclass created", utils.Bold(identity))

	if err := utils.CreateService(ctx, identity); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
	}
	glog.Infof("'%s' service created", utils.Bold(identity))

	// if err := util.CreateRBACRoles(ctx, kClient, name, identity); err != nil {
	// 	return err
	// }
	// if err := util.CreateDaemonSet(ctx, kClient, name, identity, c.kubeletDirPath, c.csiRootPath); err != nil {
	// 	return err
	// }
	// fmt.Println("Created DaemonSet", name)

	// if err := util.CreateDeployment(ctx, kClient, name, identity, c.kubeletDirPath, c.csiRootPath); err != nil {
	// 	return err
	// }
	// fmt.Println("Created Deployment", name)

	return nil
}
