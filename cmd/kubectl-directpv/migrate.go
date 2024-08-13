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
	"os"
	"strings"
	"time"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
)

var retainFlag bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate drives and volumes from legacy DirectCSI",
	Example: strings.ReplaceAll(
		`1. Migrate drives and volumes from legacy DirectCSI
   $ kubectl {PLUGIN_NAME} migrate`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, _ []string) {
		migrateMain(c.Context())
	},
}

func init() {
	setFlagOpts(migrateCmd)

	addDryRunFlag(migrateCmd, "Run in dry run mode")
	migrateCmd.PersistentFlags().BoolVar(&retainFlag, "retain", retainFlag, "retain legacy CRD after migration")
}

func migrateMain(ctx context.Context) {
	suffix := time.Now().Format(time.RFC3339)
	if err := adminClient.Migrate(ctx, admin.MigrateArgs{
		Quiet:             quietFlag,
		Retain:            retainFlag,
		DrivesBackupFile:  "directcsidrives-" + suffix + ".yaml",
		VolumesBackupFile: "directcsivolumes-" + suffix + ".yaml",
	}); err != nil {
		eprintf(true, "migration failed; %v", err)
		os.Exit(1)
	}
}
