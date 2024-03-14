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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/installer"
	"github.com/minio/directpv/pkg/utils"
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
	if err := installer.Migrate(ctx, &installer.Args{
		Quiet:  quietFlag,
		Legacy: true,
	}, false); err != nil {
		utils.Eprintf(quietFlag, true, "migration failed; %v", err)
		os.Exit(1)
	}

	if !quietFlag {
		fmt.Println("Migration successful; Please restart the pods in '" + consts.AppName + "' namespace.")
	}

	if retainFlag {
		return
	}

	suffix := time.Now().Format(time.RFC3339)

	drivesBackupFile := "directcsidrives-" + suffix + ".yaml"
	backupCreated, err := installer.RemoveLegacyDrives(ctx, drivesBackupFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to remove legacy drive CRDs; %v", err)
		os.Exit(1)
	}
	if backupCreated && !quietFlag {
		fmt.Println("Legacy drive CRDs backed up to", drivesBackupFile)
	}

	volumesBackupFile := "directcsivolumes-" + suffix + ".yaml"
	backupCreated, err = installer.RemoveLegacyVolumes(ctx, volumesBackupFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to remove legacy volume CRDs; %v", err)
		os.Exit(1)
	}
	if backupCreated && !quietFlag {
		fmt.Println("Legacy volume CRDs backed up to", volumesBackupFile)
	}
}
