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
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:           "uninstall",
	Short:         "Uninstall " + consts.AppPrettyName + " in Kubernetes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(c *cobra.Command, args []string) {
		if forceFlag {
			input := getInput(color.HiRedString("Force removal may cause data loss. Type 'Yes' if you really want to do: "))
			if input != "Yes" {
				utils.Eprintf(quietFlag, false, "Aborting...\n")
				os.Exit(1)
			}
		}

		uninstallMain(c.Context())
	},
}

func init() {
	uninstallCmd.Flags().SortFlags = false
	uninstallCmd.InheritedFlags().SortFlags = false
	uninstallCmd.LocalFlags().SortFlags = false
	uninstallCmd.LocalNonPersistentFlags().SortFlags = false
	uninstallCmd.NonInheritedFlags().SortFlags = false
	uninstallCmd.PersistentFlags().SortFlags = false

	uninstallCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "If present, uninstall forcefully")
	uninstallCmd.PersistentFlags().MarkHidden("force")
}

func uninstallMain(ctx context.Context) {
	if err := installer.Uninstall(ctx, quietFlag, forceFlag); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	if !quietFlag {
		fmt.Println(consts.AppPrettyName + " uninstalled successfully")
	}
}
