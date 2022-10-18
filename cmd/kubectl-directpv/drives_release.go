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

var drivesReleaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Release drives.",
	Example: strings.ReplaceAll(
		`# Release all the drives from all the nodes
$ kubectl {PLUGIN_NAME} drives release --all

# Release all the drives from a particular node
$ kubectl {PLUGIN_NAME} drives release --node=node1

# Release specific drives from specified nodes
$ kubectl {PLUGIN_NAME} drives release --node=node1,node2 --drive=/dev/nvme0n1

# Release specific drives from all the nodes filtered by drive ellipsis
$ kubectl {PLUGIN_NAME} drives release --drive=/dev/sd{a...b}

# Release all the drives from specific nodes filtered by node ellipsis
$ kubectl {PLUGIN_NAME} drives release --node=node{0...3}

# Release specific drives from specific nodes filtered by the combination of node and drive ellipsis
$ kubectl {PLUGIN_NAME} drives release --drive /dev/xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, _ []string) {
		if !allFlag && len(driveArgs) == 0 && len(nodeArgs) == 0 && len(accessTierArgs) == 0 {
			eprintf("atleast one of --all, --drive, --node, or --access-tier flag must be specified", true)
			os.Exit(-1)
		}
		drivesReleaseMain(c.Context())
	},
}

func init() {
	drivesReleaseCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "Release all drives on all nodes.")
}

func drivesReleaseMain(ctx context.Context) {
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

	var drivesProcessed bool
	var releasedDrives, failedDrives []string
	for result := range resultCh {
		if result.Err != nil {
			eprintf(result.Err.Error(), true)
			os.Exit(1)
		}

		drivesProcessed = true

		switch result.Drive.Status.Status {
		case directpvtypes.DriveStatusReleased:
			releasedDrives = append(
				releasedDrives,
				fmt.Sprintf("%v/%v", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			)
		default:
			var errMsg string
			volumeCount := result.Drive.GetVolumeCount()
			if volumeCount > 0 {
				errMsg = fmt.Sprintf(
					"%v/%v: %v volumes still exist",
					result.Drive.GetNodeID(),
					result.Drive.GetDriveName(),
					volumeCount,
				)
			} else {
				result.Drive.Status.Status = directpvtypes.DriveStatusReleased
				if !dryRun {
					if _, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil {
						errMsg = err.Error()
					}
				}
			}

			if errMsg == "" {
				releasedDrives = append(
					releasedDrives,
					fmt.Sprintf("%v/%v", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
				)
			} else {
				failedDrives = append(failedDrives, errMsg)
			}
		}
	}

	if drivesProcessed {
		if len(releasedDrives) != 0 {
			fmt.Println("Released drives:")
			fmt.Println(strings.Join(releasedDrives, "\n"))
		}

		if len(failedDrives) != 0 {
			for _, failedDrive := range failedDrives {
				eprintf(failedDrive, true)
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
