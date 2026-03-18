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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:           "remove [DRIVE ...]",
	Short:         "Remove unused drives from " + consts.AppPrettyName,
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Remove an unused drive from all nodes
   $ kubectl {PLUGIN_NAME} remove --drives=nvme1n1

2. Remove all unused drives from a node
   $ kubectl {PLUGIN_NAME} remove --nodes=node1

3. Remove specific unused drives from specific nodes
   $ kubectl {PLUGIN_NAME} remove --nodes=node{1...4} --drives=sd{a...f}

4. Remove all unused drives from all nodes
   $ kubectl {PLUGIN_NAME} remove --all

5. Remove drives are in 'error' status
   $ kubectl {PLUGIN_NAME} remove --status=error`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateRemoveCmd(); err != nil {
			eprintf(true, "%v\n", err)
			os.Exit(-1)
		}

		removeMain(c.Context())
	},
}

func init() {
	setFlagOpts(removeCmd)

	addNodesFlag(removeCmd, "If present, select drives from given nodes")
	addDrivesFlag(removeCmd, "If present, select drives by given names")
	addDriveStatusFlag(removeCmd, "If present, select drives by drive status")
	addAllFlag(removeCmd, "If present, select all unused drives")
	addDryRunFlag(removeCmd, "Run in dry run mode")
}

func validateRemoveCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}

	if err := validateDriveNameArgs(); err != nil {
		return err
	}

	if err := validateDriveStatusArgs(); err != nil {
		return err
	}

	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	switch {
	case allFlag:
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(driveStatusArgs) != 0:
	case len(driveIDArgs) != 0:
	default:
		return errors.New("no drive selected to remove")
	}

	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveStatusSelectors = nil
		driveIDSelectors = nil
	}

	return nil
}

func removeMain(ctx context.Context) {
	_, err := adminClient.Remove(
		ctx,
		admin.RemoveArgs{
			Nodes:       nodesArgs,
			Drives:      drivesArgs,
			DriveStatus: driveStatusSelectors,
			DriveIDs:    driveIDSelectors,
			DryRun:      dryRunFlag,
		},
		logFunc,
	)
	if err != nil {
		eprintf(!errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
