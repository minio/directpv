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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var labelDrivesCmd = &cobra.Command{
	Use:           "drives k=v|k-",
	Aliases:       []string{"drive", "dr"},
	Short:         "Set labels to drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Set 'tier: hot' label to all drives in all nodes
   $ kubectl {PLUGIN_NAME} label drives tier=hot --all

2. Set 'type: fast' to specific drives from a node
   $ kubectl {PLUGIN_NAME} label drives type=fast --nodes=node1 --drives=nvme1n{1...3}

3. Remove 'tier: hot' label from all drives in all nodes
   $ kubectl {PLUGIN_NAME} label drives tier- --all`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = idArgs
		if err := validateLabelDrivesCmd(args); err != nil {
			utils.Eprintf(quietFlag, true, "%s; Check `--help` for usage\n", err.Error())
			os.Exit(1)
		}
		labelDrivesMain(c.Context())
	},
}

func validateLabelDrivesCmd(args []string) (err error) {
	if err = validateLabelArgs(); err != nil {
		return err
	}
	if err = validateListDrivesArgs(); err != nil {
		return err
	}
	switch {
	case allFlag:
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(driveStatusArgs) != 0:
	case len(driveIDArgs) != 0:
	case len(labelArgs) != 0:
	default:
		return errors.New("no drives selected to label")
	}
	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveStatusArgs = nil
		driveIDArgs = nil
		labelArgs = nil
	}
	labels, err = validateLabelCmdArgs(args)
	return err
}

func init() {
	setFlagOpts(labelDrivesCmd)

	addDriveStatusFlag(labelDrivesCmd, "If present, select drives by status")
	addIDFlag(labelDrivesCmd, "If present, select by drive ID")
	addLabelsFlag(labelDrivesCmd, "If present, select by drive labels")
}

func labelDrivesMain(ctx context.Context) {
	if err := admin.LabelDrives(ctx, admin.LabelDriveArgs{
		Nodes:          nodesArgs,
		Drives:         drivesArgs,
		DriveStatus:    driveStatusSelectors,
		DriveIDs:       driveIDSelectors,
		LabelSelectors: labelSelectors,
		Quiet:          quietFlag,
		DryRun:         dryRunFlag,
	}, labels); err != nil {
		utils.Eprintf(quietFlag, !errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
