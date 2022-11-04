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
	"os"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var crdFlag bool

var uninstallCmd = &cobra.Command{
	Use:          "uninstall",
	Short:        "Uninstall " + consts.AppPrettyName + " in Kubernetes.",
	SilenceUsage: true,
	Run: func(c *cobra.Command, args []string) {
		if crdFlag || forceFlag {
			input := getInput(color.HiRedString("CRD removal may cause data loss. Type 'Yes' if you really want to do: "))
			if input != "Yes" {
				utils.Eprintf(quietFlag, false, "Aborting...\n")
				os.Exit(1)
			}
		}

		uninstallMain(c.Context())
	},
}

func init() {
	uninstallCmd.PersistentFlags().BoolVar(&crdFlag, "crd", crdFlag, "If present, remove CRDs")
	uninstallCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "If present, uninstall forcefully")
	uninstallCmd.PersistentFlags().MarkHidden("crd")
	uninstallCmd.PersistentFlags().MarkHidden("force")
}

func uninstallMain(ctx context.Context) {
	installConfig := &installer.Config{
		Identity:     consts.Identity,
		UninstallCRD: crdFlag,
		ForceRemove:  forceFlag,
	}

	if err := installer.Uninstall(ctx, installConfig); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if !quietFlag {
		color.Red("\n%s is uninstalled successfully", consts.AppPrettyName)
	}
}
