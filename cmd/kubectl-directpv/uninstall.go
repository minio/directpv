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
	Run: func(c *cobra.Command, _ []string) {
		uninstallMain(c.Context())
	},
}

func init() {
	setFlagOpts(uninstallCmd)

	addDangerousFlag(uninstallCmd, "If present, uninstall forcefully which may cause data loss")
	uninstallCmd.PersistentFlags().MarkHidden("dangerous")
}

func uninstallMain(ctx context.Context) {
	if err := installer.Uninstall(ctx, quietFlag, dangerousFlag); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	if !quietFlag {
		fmt.Println(consts.AppPrettyName + " uninstalled successfully")
	}
}
