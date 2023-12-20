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

var cordonCmd = &cobra.Command{
	Use:           "cordon [DRIVE ...]",
	Short:         "Mark drives as unschedulable",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Cordon all drives from all nodes
   $ kubectl {PLUGIN_NAME} cordon --all

2. Cordon all drives from a node
   $ kubectl {PLUGIN_NAME} cordon --nodes=node1

3. Cordon a drive from all nodes
   $ kubectl {PLUGIN_NAME} cordon --drives=nvme1n1

4. Cordon specific drives from specific nodes
   $ kubectl {PLUGIN_NAME} cordon --nodes=node{1...4} --drives=sd{a...f}

5. Cordon drives which are in 'error' status
   $ kubectl {PLUGIN_NAME} cordon --status=error`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateCordonCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		cordonMain(c.Context())
	},
}

func init() {
	setFlagOpts(cordonCmd)

	addNodesFlag(cordonCmd, "If present, select drives from given nodes")
	addDrivesFlag(cordonCmd, "If present, select drives by given names")
	addDriveStatusFlag(cordonCmd, "If present, select drives by drive status")
	addAllFlag(cordonCmd, "If present, select all drives")
	addDryRunFlag(cordonCmd, "Run in dry run mode")
}

func validateCordonCmd() error {
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
		return errors.New("no drive selected to cordon")
	}

	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveStatusSelectors = nil
		driveIDSelectors = nil
	}

	return nil
}

func cordonMain(ctx context.Context) {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		processed = true

		if result.Drive.IsUnschedulable() {
			continue
		}

		volumes := result.Drive.GetVolumes()
		if len(volumes) != 0 {
			for vresult := range client.NewVolumeLister().VolumeNameSelector(volumes).IgnoreNotFound(true).List(ctx) {
				if vresult.Err != nil {
					utils.Eprintf(quietFlag, true, "%v\n", vresult.Err)
					os.Exit(1)
				}

				if vresult.Volume.Status.Status == directpvtypes.VolumeStatusPending {
					utils.Eprintf(quietFlag, true, "unable to cordon drive %v; pending volumes found\n", result.Drive.GetDriveID())
					os.Exit(1)
				}
			}
		}

		result.Drive.Unschedulable()
		if !dryRunFlag {
			if _, err := client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil {
				utils.Eprintf(quietFlag, true, "unable to cordon drive %v; %v\n", result.Drive.GetDriveID(), err)
				os.Exit(1)
			}
		}

		if !quietFlag {
			fmt.Printf("Drive %v/%v cordoned\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
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
}
