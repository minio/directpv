// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

	"github.com/spf13/cobra"

	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

var uninstallCmd = &cobra.Command{
	Use:          "uninstall",
	Short:        utils.BinaryNameTransform("Uninstall {{ . }} in k8s cluster"),
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
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "unregister direct.csi.min.io group crds [May cause data loss]")
	uninstallCmd.PersistentFlags().BoolVarP(&forceRemove, "force", "", forceRemove, "Removes the direct.csi.min.io resources [May cause data loss]")

	uninstallCmd.PersistentFlags().MarkHidden("crd")
	uninstallCmd.PersistentFlags().MarkHidden("force")
}

func uninstall(ctx context.Context, args []string) error {
	if dryRun {
		klog.Errorf("'--dry-run' flag is not supported for uninstall")
		return nil
	}

	installConfig := &installer.Config{
		Identity:     identity,
		UninstallCRD: uninstallCRD,
		ForceRemove:  forceRemove,
	}

	return installer.Uninstall(ctx, installConfig)
}
