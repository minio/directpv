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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var drivesSetStatusCmd = &cobra.Command{
	Use:    fmt.Sprintf("status <%v>", strings.Join(driveStatusValues(), "|")),
	Short:  "Set status to drives.",
	Hidden: true,
	Example: strings.ReplaceAll(
		`# Set all the drives as OK status
$ kubectl {PLUGIN_NAME} drives set status ok --all

# Set all the drives from particular node as OK status
$ kubectl {PLUGIN_NAME} drives set status ok --node=node1

# Set specified drives from specified nodes as error status
$ kubectl {PLUGIN_NAME} drives set status error --node=node1,node2 --drive=/dev/nvme0n1

# Set drives filtered by specified drive ellipsis as error status
$ kubectl {PLUGIN_NAME} drives set status error --drive=/dev/sd{a...b}

# Set drives filtered by specified node ellipsis as deleted
$ kubectl {PLUGIN_NAME} drives set status deleted --node=node{0...3}

# Set drives filtered by specified combination of node and drive ellipsis as lost status
$ kubectl {PLUGIN_NAME} drives set status lost --drive /dev/xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if !allFlag && len(driveArgs) == 0 && len(nodeArgs) == 0 {
			eprintf("atleast one of --all, --drive or --node must be provided", true)
			os.Exit(-1)
		}
		if len(args) != 1 {
			eprintf("only one status must be provided", true)
			os.Exit(-1)
		}
		if !utils.Contains(driveStatusValues(), args[0]) {
			eprintf(fmt.Sprintf("unknown drive status %v", args[0]), true)
			os.Exit(-1)
		}
		status, err := directpvtypes.ToDriveStatus(args[0])
		if err != nil {
			eprintf(fmt.Sprintf("unable to convert to drive status; %v", err), true)
			os.Exit(-1)
		}
		drivesSetStatusMain(c.Context(), status)
	},
}

func drivesSetStatusMain(ctx context.Context, status directpvtypes.DriveStatus) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	resultCh, err := drive.ListDrives(ctx, nodeSelectors, driveSelectors, accessTierSelectors, driveStatusArgs, k8s.MaxThreadCount)
	if err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}

	var drivesProcessed bool
	var processedDrives, failedDrives []string
	for result := range resultCh {
		if result.Err != nil {
			eprintf(result.Err.Error(), true)
			os.Exit(1)
		}

		drivesProcessed = true
		switch {
		case result.Drive.Status.Status == status:
			processedDrives = append(
				processedDrives,
				fmt.Sprintf("%v/%v", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			)
		default:
			var errMsg string
			result.Drive.Status.Status = status
			if !dryRun {
				if _, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil {
					errMsg = err.Error()
				}
			}
			if errMsg == "" {
				processedDrives = append(
					processedDrives,
					fmt.Sprintf("%v/%v", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
				)
			} else {
				failedDrives = append(failedDrives, errMsg)
			}
		}
	}

	if drivesProcessed {
		if len(processedDrives) != 0 {
			fmt.Println("Processed drives:")
			fmt.Println(strings.Join(processedDrives, "\n"))
		}

		if len(failedDrives) != 0 {
			for _, failedDrive := range failedDrives {
				eprintf(failedDrive, false)
			}
			os.Exit(1)
		}

		return
	}

	if allFlag {
		eprintf("No resources found", false)
	} else {
		eprintf("No matching resources found", false)
	}

	os.Exit(1)
}
