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
	"io"

	"github.com/minio/kubectl-direct-csi/util"
	"github.com/spf13/cobra"
)

const (
	csiInstallDesc = `
'install' command creates MinIO Direct CSI along with all the dependencies.`
	csiInstallExample = `  kubectl direct-csi install`
)

type csiInstallCmd struct {
	out            io.Writer
	errOut         io.Writer
	output         bool
	kubeletDirPath string
	csiRootPath    string
}

func newInstallCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	c := &csiInstallCmd{out: out, errOut: errOut, kubeletDirPath: "/var/lib/kubelet", csiRootPath: "/mnt/direct-csi"}

	cmd := &cobra.Command{
		Use:     "install",
		Short:   "Install MinIO Direct CSI",
		Long:    csiInstallDesc,
		Example: csiInstallExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.run()
		},
	}

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (c *csiInstallCmd) run() error {
	name := "direct-csi-min-io"
	identity := "direct-csi-min-io"
	ctx := context.Background()

	kClient := util.GetKubeClient()

	if err := util.CreateDirectCSINamespace(ctx, kClient, name); err != nil {
		return err
	}
	if err := util.CreateCSIDriver(ctx, kClient, name); err != nil {
		return err
	}
	if err := util.CreateStorageClass(ctx, kClient, name); err != nil {
		return err
	}
	if err := util.CreateCSIService(ctx, kClient, name); err != nil {
		return err
	}
	if err := util.CreateDaemonSet(ctx, kClient, name, identity, c.kubeletDirPath, c.csiRootPath); err != nil {
		return err
	}
	if err := util.CreateDeployment(ctx, kClient, name, identity, c.kubeletDirPath, c.csiRootPath); err != nil {
		return err
	}
	return nil
}
