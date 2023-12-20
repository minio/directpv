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
	"fmt"
	"os"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var removeCmd = &cobra.Command{
	Use:           "remove [DRIVE ...]",
	Short:         fmt.Sprintf("Remove unused drives from %s", consts.AppPrettyName),
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
			utils.Eprintf(quietFlag, true, "%v\n", err)
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
	var processed bool
	var failed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		processed = true
		switch result.Drive.Status.Status {
		case directpvtypes.DriveStatusRemoved:
		default:
			volumeCount := result.Drive.GetVolumeCount()
			if volumeCount > 0 {
				failed = true
			} else {
				result.Drive.Status.Status = directpvtypes.DriveStatusRemoved
				var err error
				if !dryRunFlag {
					_, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{})
				}
				if err != nil {
					failed = true
					utils.Eprintf(quietFlag, true, "%v/%v: %v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName(), err)
				} else if !quietFlag {
					fmt.Printf("Removing %v/%v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
				}
			}
		}
	}

	if !processed {
		if allFlag {
			utils.Eprintf(quietFlag, false, "No resources found\n")
		} else {
			utils.Eprintf(quietFlag, false, "No matching resources found\n")
		}

		os.Exit(1)
	}

	if failed {
		os.Exit(1)
	}
}
