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

var uncordonDrivesCmd = &cobra.Command{
	Use:   "uncordon",
	Short: "Uncordon drives to resume scheduling of new volumes",
	Example: strings.ReplaceAll(
		`# Uncordon all the drives from all the nodes
$ kubectl {PLUGIN_NAME} drives uncordon --all

# Uncordon all the drives from a particular node
$ kubectl {PLUGIN_NAME} drives uncordon --node=node1

# Uncordon specific drives from specified nodes
$ kubectl {PLUGIN_NAME} drives uncordon --node=node1,node2 --drive=/dev/nvme0n1

# Uncordon specific drives from all the nodes filtered by drive ellipsis
$ kubectl {PLUGIN_NAME} drives uncordon --drive=/dev/sd{a...b}

# Uncordon all the drives from specific nodes filtered by node ellipsis
$ kubectl {PLUGIN_NAME} drives uncordon --node=node{0...3}

# Uncordon specific drives from specific nodes filtered by the combination of node and drive ellipsis
$ kubectl {PLUGIN_NAME} drives uncordon --drive /dev/xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	RunE: func(c *cobra.Command, _ []string) error {
		if !allFlag {
			if len(driveArgs) == 0 && len(nodeArgs) == 0 && len(accessTierArgs) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s', '%s' or '%s' must be specified",
					bold("--all"),
					bold("--drive"),
					bold("--node"),
					bold("--access-tier"),
				)
			}
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		return uncordonDrives(c.Context())
	},
}

func init() {
	uncordonDrivesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Uncordon drives filtered by drive path (supports ellipses pattern)")
	uncordonDrivesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Uncordon drives filtered by nodes (supports ellipses pattern)")
	uncordonDrivesCmd.PersistentFlags().StringSliceVarP(&accessTierArgs, "access-tier", "", accessTierArgs, fmt.Sprintf("Uncordon drives filtered by access tier set on the drive [%s]", strings.Join(directpvtypes.SupportedAccessTierValues(), ", ")))
	uncordonDrivesCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "Uncordon all the drives from all the nodes")
}

func uncordonDrives(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	resultCh, err := drive.ListDrives(ctx, nodeSelectors, driveSelectors, accessTierSelectors, k8s.MaxThreadCount)
	if err != nil {
		return err
	}
	return drive.ProcessDrives(
		ctx,
		resultCh,
		func(drive *types.Drive) bool {
			return drive.Status.Status == directpvtypes.DriveStatusCordoned
		},
		func(drive *types.Drive) error {
			drive.Status.Status = directpvtypes.DriveStatusOK
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
