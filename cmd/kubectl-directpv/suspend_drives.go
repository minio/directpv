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
	"fmt"
	"os"
	"strings"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

var suspendDrivesCmd = &cobra.Command{
	Use:           "drives [DRIVE ...]",
	Short:         "Suspend drives",
	Long:          "Suspend the drives (CAUTION: This will make the corresponding volumes as read-only)",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Suspend all drives from a node
   $ kubectl {PLUGIN_NAME} suspend drives --nodes=node1

2. Suspend specific drive from specific node
   $ kubectl {PLUGIN_NAME} suspend drives --nodes=node1 --drives=sda

3. Suspend a drive by its DRIVE-ID 'af3b8b4c-73b4-4a74-84b7-1ec30492a6f0'
   $ kubectl {PLUGIN_NAME} suspend drives af3b8b4c-73b4-4a74-84b7-1ec30492a6f0`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateSuspendDrivesCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		if !dangerousFlag {
			utils.Eprintf(quietFlag, true, "Suspending the drives will make the corresponding volumes as read-only. Please review carefully before performing this *DANGEROUS* operation and retry this command with --dangerous flag..\n")
			os.Exit(1)
		}

		suspendDrivesMain(c.Context())
	},
}

func init() {
	setFlagOpts(suspendDrivesCmd)

	addNodesFlag(suspendDrivesCmd, "If present, suspend drives from given nodes")
	addDrivesFlag(suspendDrivesCmd, "If present, suspend drives by given names")
	addDangerousFlag(suspendDrivesCmd, "Suspending the drives will make the corresponding volumes as read-only")
}

func validateSuspendDrivesCmd() error {
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
		return errors.New("no drive selected to suspend")
	}

	return nil
}

func suspendDrivesMain(ctx context.Context) {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		DriveIDSelector(driveIDSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		processed = true

		if result.Drive.IsSuspended() {
			continue
		}

		driveClient := client.DriveClient()
		updateFunc := func() error {
			drive, err := driveClient.Get(ctx, result.Drive.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			drive.Suspend()
			if !dryRunFlag {
				if _, err := driveClient.Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			utils.Eprintf(quietFlag, true, "unable to suspend drive %v; %v\n", result.Drive.GetDriveID(), err)
			os.Exit(1)
		}

		if !quietFlag {
			fmt.Printf("Drive %v/%v suspended\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
		}
	}

	if !processed {
		utils.Eprintf(quietFlag, false, "No matching resources found\n")
		os.Exit(1)
	}
}
