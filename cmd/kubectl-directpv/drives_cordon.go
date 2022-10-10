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
	"k8s.io/klog/v2"
)

var cordonDrivesCmd = &cobra.Command{
	Use:   "cordon",
	Short: "Cordon drive(s) to prevent any new volumes to be scheduleed",
	Example: strings.ReplaceAll(
		`# Cordon all the drives from all the nodes
	$ kubectl {PLUGIN_NAME} drives cordon --all

	# Cordon all the drives from a particular node
	$ kubectl {PLUGIN_NAME} drives cordon --node=node1

	# Cordon specific drives from specified nodes
	$ kubectl {PLUGIN_NAME} drives cordon --node=node1,node2 --drive=/dev/nvme0n1

	# Cordon specific drives from all the nodes filtered by drive ellipsis
	$ kubectl {PLUGIN_NAME} drives cordon --drive=/dev/sd{a...b}

	# Cordon all the drives from specific nodes filtered by node ellipsis
	$ kubectl {PLUGIN_NAME} drives cordon --node=node{0...3}

	# Cordon specific drives from specific nodes filtered by the combination of node and drive ellipsis
	$ kubectl {PLUGIN_NAME} drives cordon --drive /dev/xvd{a...d} --node node{1...4}`,
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
		return cordonDrives(c.Context())
	},
}

func init() {
	cordonDrivesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter drives to be cordoned by drive path (supports ellipses pattern)")
	cordonDrivesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter drives to be cordoned by nodes (supports ellipses pattern)")
	cordonDrivesCmd.PersistentFlags().StringSliceVarP(&accessTierArgs, "access-tier", "", accessTierArgs, fmt.Sprintf("Filter drives to be cordoned by access tier set on the drive [%s]", strings.Join(directpvtypes.SupportedAccessTierValues(), ", ")))
	cordonDrivesCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "Cordon all the drives from all the nodes")
}

func cordonDrives(ctx context.Context) error {
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
			if drive.Status.Status == directpvtypes.DriveStatusCordoned {
				klog.Errorf("%s is already cordoned", bold(drive.Status.Path))
				return false
			}
			if drive.Status.Status != directpvtypes.DriveStatusOK {
				klog.Errorf("%s is in %s state. only %s drives can be cordoned", bold(drive.Status.Path), bold(drive.Status.Status), bold(directpvtypes.DriveStatusOK))
				return false
			}
			return true
		},
		func(drive *types.Drive) error {
			drive.Status.Status = directpvtypes.DriveStatusCordoned
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
