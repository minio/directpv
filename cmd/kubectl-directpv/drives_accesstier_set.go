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
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var accessTierSetCmd = &cobra.Command{
	Use:   "set [hot|cold|warm]",
	Short: "Set accesstiers on the drive(s) added by " + consts.AppPrettyName,
	Example: strings.ReplaceAll(
		`# Set all the drives as hot tiered
$ kubectl {PLUGIN_NAME} drives access-tier set hot --all

# Set all the drives from particular node as cold tiered
$ kubectl {PLUGIN_NAME} drives access-tier set cold --node=node1

# Set specified drives from specified nodes as warm tiered
$ kubectl {PLUGIN_NAME} drives access-tier set warm --node=node1,node2 --drive=/dev/nvme0n1

# Set drives filtered by specified drive ellipsis as cold tiered
$ kubectl {PLUGIN_NAME} drives access-tier set cold --drive=/dev/sd{a...b}

# Set drives filtered by specified node ellipsis as hot tiered
$ kubectl {PLUGIN_NAME} drives access-tier set hot --node=node{0...3}

# Set drives filtered by specified combination of node and drive ellipsis as cold tiered
$ kubectl {PLUGIN_NAME} drives access-tier set cold --drive /dev/xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	RunE: func(c *cobra.Command, args []string) error {
		if !allFlag {
			if len(driveArgs) == 0 && len(nodeArgs) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s' or '%s' must be specified",
					bold("--all"),
					bold("--drive"),
					bold("--node"),
				)
			}
		}
		if len(args) != 1 {
			return fmt.Errorf("Invalid syntax. Please use '%s' for examples to set access-tier", bold("--help"))
		}
		accessTier, err := directpvtypes.ToAccessTier(args[0])
		if err != nil {
			return err
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		return setAccessTier(c.Context(), accessTier)
	},
}

func init() {
	accessTierSetCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter by drive paths for setting access-tier (supports ellipses pattern)")
	accessTierSetCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter by nodes for setting access-tier (supports ellipses pattern)")
	accessTierSetCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "Set provided access-tier on all the drives from all the nodes")
}

func setAccessTier(ctx context.Context, accessTier directpvtypes.AccessTier) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	resultCh, err := drive.ListDrives(ctx, nodeSelectors, driveSelectors, nil, k8s.MaxThreadCount)
	if err != nil {
		return err
	}
	return drive.ProcessDrives(
		ctx,
		resultCh,
		func(drive *types.Drive) bool {
			return drive.Status.AccessTier != accessTier
		},
		func(drive *types.Drive) error {
			drive.Status.AccessTier = accessTier
			return nil
		},
		func(ctx context.Context, drive *types.Drive) error {
			_, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{})
			return err
		},
		nil,
		dryRun,
	)
}
