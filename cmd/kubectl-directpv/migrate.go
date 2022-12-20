// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate drives and volumes from legacy DirectCSI",
	Example: strings.ReplaceAll(
		`# Migrate drives and volumes from legacy DirectCSI
$ kubectl {PLUGIN_NAME} migrate`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		migrateMain(c.Context())
	},
}

func init() {
	migrateCmd.Flags().SortFlags = false
	migrateCmd.InheritedFlags().SortFlags = false
	migrateCmd.LocalFlags().SortFlags = false
	migrateCmd.LocalNonPersistentFlags().SortFlags = false
	migrateCmd.NonInheritedFlags().SortFlags = false
	migrateCmd.PersistentFlags().SortFlags = false

	addDryRunFlag(migrateCmd)
}

func migrateMain(ctx context.Context) {
	auditFile := fmt.Sprintf("migrate.log.%v", time.Now().UTC().Format(time.RFC3339Nano))
	file, err := openAuditFile(auditFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to open audit file %v; %v\n", auditFile, err)
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("Skipping audit logging"))
	}

	defer func() {
		if file != nil {
			if err := file.Close(); err != nil {
				utils.Eprintf(quietFlag, true, "unable to close audit file; %v\n", err)
			}
		}
	}()

	if err := installer.Migrate(ctx, &installer.MigrateArgs{
		DryRun:      dryRunFlag,
		AuditWriter: file,
		Quiet:       quietFlag,
		Progress:    nil,
	}); err != nil {
		utils.Eprintf(quietFlag, true, "migration failed; %v", err)
		os.Exit(1)
	}

	if !quietFlag {
		fmt.Println("Migration successful")
	}
}
