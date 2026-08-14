// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	forceFlag           = false
	disablePrefetchFlag = false
)

var repairCmd = &cobra.Command{
	Use:           "repair DRIVE ...",
	Short:         "Repair filesystem of drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Repair drives
   $ kubectl {PLUGIN_NAME} repair 3b562992-f752-4a41-8be4-4e688ae8cd4c

2. Generate repair job manifest of drives without creating them
   $ kubectl {PLUGIN_NAME} repair 3b562992-f752-4a41-8be4-4e688ae8cd4c -o yaml`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args
		if err := validateRepairCmd(); err != nil {
			eprintf(true, "%v\n", err)
			os.Exit(-1)
		}

		repairMain(c.Context())
	},
}

func init() {
	setFlagOpts(repairCmd)

	addDryRunFlag(repairCmd, "Repair drives with no modify mode")
	repairCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "Force log zeroing")
	repairCmd.PersistentFlags().BoolVar(&disablePrefetchFlag, "disable-prefetch", disablePrefetchFlag, "Disable prefetching of inode and directory blocks")
	addOutputFormatFlag(repairCmd, "Generate repair job manifest of drives. One of: yaml|json")
}

func validateRepairCmd() error {
	if err := validateOutputFormat(false); err != nil {
		return err
	}

	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	if len(driveIDArgs) == 0 {
		return errors.New("no drive provided to repair")
	}

	return nil
}

func repairMain(ctx context.Context) {
	args := admin.RepairArgs{
		DriveIDs:            driveIDSelectors,
		DryRun:              dryRunFlag,
		ForceFlag:           forceFlag,
		DisablePrefetchFlag: disablePrefetchFlag,
	}

	if dryRunPrinter != nil {
		jobs, err := adminClient.GetRepairJobs(ctx, args, logFunc)
		if err != nil {
			eprintf(!errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
			os.Exit(1)
		}

		if len(jobs) == 0 {
			eprintf(false, "No matching resources found\n")
			os.Exit(1)
		}

		dryRunPrinter(batchv1.JobList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			Items: jobs,
		})
		return
	}

	_, err := adminClient.Repair(ctx, args, logFunc)
	if err != nil {
		eprintf(!errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
