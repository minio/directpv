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
	"errors"
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:          "uninstall",
	Short:        "Uninstall " + consts.AppPrettyName + " in kubernetes cluster",
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
	uninstallCmd.PersistentFlags().BoolVarP(&uninstallCRD, "crd", "c", uninstallCRD, "unregister "+consts.GroupName+" group crds [May cause data loss]")
	uninstallCmd.PersistentFlags().BoolVarP(&forceRemove, "force", "", forceRemove, "Removes the "+consts.GroupName+" resources [May cause data loss]")

	uninstallCmd.PersistentFlags().MarkHidden("crd")
	uninstallCmd.PersistentFlags().MarkHidden("force")
}

func uninstall(ctx context.Context, args []string) error {
	if dryRun {
		return errors.New("uninstall does not support --dry-run flag")
	}

	installConfig := &installer.Config{
		Identity:     identity,
		UninstallCRD: uninstallCRD,
		ForceRemove:  forceRemove,
	}

	err := installer.Uninstall(ctx, installConfig)
	if err == nil {
		fmt.Println(color.HiWhiteString(consts.AppPrettyName), "is uninstalled successfully")
	}
	return err
}
