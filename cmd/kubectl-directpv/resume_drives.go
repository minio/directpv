// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022, 2023 MinIO, Inc.
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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var resumeDrivesCmd = &cobra.Command{
	Use:           "drives [DRIVE ...]",
	Short:         "Resume suspended drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Resume all suspended drives from a node
   $ kubectl {PLUGIN_NAME} resume drives --nodes=node1

2. Resume specific drive from specific node
   $ kubectl {PLUGIN_NAME} resume drives --nodes=node1 --drives=sda

3. Resume a suspended drive by its DRIVE-ID 'af3b8b4c-73b4-4a74-84b7-1ec30492a6f0'
   $ kubectl {PLUGIN_NAME} resume drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateResumeDrivesCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		resumeDrivesMain(c.Context())
	},
}

func init() {
	setFlagOpts(resumeDrivesCmd)

	addNodesFlag(resumeDrivesCmd, "If present, resume drives from given nodes")
	addDrivesFlag(resumeDrivesCmd, "If present, resume drives by given names")
}

func validateResumeDrivesCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}
	if err := validateDriveNameArgs(); err != nil {
		return err
	}
	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	switch {
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(driveIDArgs) != 0:
	default:
		return errors.New("no drive selected to resume")
	}

	return nil
}

func resumeDrivesMain(ctx context.Context) {
	if err := admin.ResumeDrives(ctx, admin.ResumeDriveArgs{
		Nodes:            nodesArgs,
		Drives:           drivesArgs,
		DriveIDSelectors: driveIDSelectors,
		Quiet:            quietFlag,
		DryRun:           dryRunFlag,
	}); err != nil {
		utils.Eprintf(quietFlag, !errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
