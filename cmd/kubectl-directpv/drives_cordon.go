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
	"os"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var drivesCordonCmd = &cobra.Command{
	Use:   "cordon",
	Short: "Cordon drives.",
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
	Run: func(c *cobra.Command, _ []string) {
		if !allFlag && len(driveArgs) == 0 && len(nodeArgs) == 0 && len(accessTierArgs) == 0 {
			eprintf("atleast one of --all, --drive, --node, or --access-tier flag must be specified", true)
			os.Exit(-1)
		}
		drivesCordonMain(c.Context())
	},
}

func init() {
	drivesCordonCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "Cordon all drives on all nodes.")
}

func drivesCordonMain(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var err error
	var resultCh <-chan drive.ListDriveResult
	if allFlag {
		resultCh, err = drive.ListDrives(ctx, nil, nil, nil, nil, k8s.MaxThreadCount)
	} else {
		resultCh, err = drive.ListDrives(ctx, nodeSelectors, driveSelectors, accessTierSelectors, driveStatusArgs, k8s.MaxThreadCount)
	}
	if err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}

	var processed bool
	for result := range resultCh {
		if result.Err != nil {
			eprintf(result.Err.Error(), true)
			os.Exit(1)
		}

		processed = true

		if result.Drive.IsUnschedulable() {
			fmt.Printf("Drive %v already cordoned\n", result.Drive.GetDriveID())
			continue
		}

		for volumeResult := range getVolumesByNames(ctx, result.Drive.GetVolumes(), true) {
			if volumeResult.Err != nil {
				eprintf(result.Err.Error(), true)
				os.Exit(1)
			}

			if volumeResult.Volume.GetStatus() == directpvtypes.VolumeStatusPending {
				eprintf(fmt.Sprintf("unable to cordon drive %v; pending volumes found", result.Drive.GetDriveID()), true)
				os.Exit(1)
			}
		}

		result.Drive.Unschedulable()
		if !dryRun {
			if _, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil {
				eprintf(fmt.Sprintf("unable to cordon drive %v; %v", result.Drive.GetDriveID(), err), true)
				os.Exit(1)
			}
		}

		fmt.Printf("Drive %v cordoned\n", result.Drive.GetDriveID())
	}

	if !processed {
		if allFlag {
			eprintf("No resources found", false)
		} else {
			eprintf("No matching resources found", false)
		}

		os.Exit(1)
	}
}
