/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/kubectl-directcsi/util"
	"github.com/spf13/cobra"
)

const (
	csiUninstallDesc = `
 uninstall command deletes MinIO DirectCSI along with all the dependencies.`
	csiUninstallExample = `  kubectl directcsi uninstall`
)

type csiUninstallCmd struct {
	out            io.Writer
	errOut         io.Writer
	output         bool
	kubeletDirPath string
	csiRootPath    string
}

func newUninstallCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	c := &csiUninstallCmd{out: out, errOut: errOut, kubeletDirPath: "/var/lib/kubelet", csiRootPath: "/mnt/direct-csi"}

	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall MinIO DirectCSI",
		Long:    csiUninstallDesc,
		Example: csiUninstallExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.run()
		},
	}

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (c *csiUninstallCmd) run() error {
	name := "direct-csi-min-io"
	identity := "direct-csi"
	ctx := context.Background()

	kClient := util.GetKubeClient()

	if err := util.RemoveDirectCSINamespace(ctx, kClient, identity); err != nil {
		return err
	}
	fmt.Println("Deleted Namespace ", identity)
	if err := util.RemoveCSIDriver(ctx, kClient, name); err != nil {
		return err
	}
	fmt.Println("Deleted CSIDriver ", name)
	if err := util.RemoveStorageClass(ctx, kClient, name); err != nil {
		return err
	}
	fmt.Println("Deleted StorageClass ", name)
	if err := util.RemoveCSIService(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Deleted CSIDriver ", name)
	if err := util.RemoveRBACRoles(ctx, kClient, name, identity); err != nil {
		return err
	}
	if err := util.RemoveDaemonSet(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Deleted DaemonSet ", name)
	if err := util.RemoveDeployment(ctx, kClient, name, identity); err != nil {
		return err
	}
	fmt.Println("Deleted Deployment ", name)
	return nil
}
