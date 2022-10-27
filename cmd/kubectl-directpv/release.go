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
	"github.com/minio/directpv/pkg/drive"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var releaseCmd = &cobra.Command{
	Use:   "release [DRIVE ...]",
	Short: "Release drives.",
	Example: strings.ReplaceAll(
		`# Release all the drives from all the nodes
$ kubectl {PLUGIN_NAME} release --all

# Release all the drives from a particular node
$ kubectl {PLUGIN_NAME} release --node=node1

# Release specific drives from specified nodes
$ kubectl {PLUGIN_NAME} release --node=node1,node2 --drive=nvme0n1

# Release specific drives from all the nodes filtered by drive ellipsis
$ kubectl {PLUGIN_NAME} release --drive=sd{a...b}

# Release all the drives from specific nodes filtered by node ellipsis
$ kubectl {PLUGIN_NAME} release --node=node{0...3}

# Release specific drives from specific nodes filtered by the combination of node and drive ellipsis
$ kubectl {PLUGIN_NAME} release --drive xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateReleaseCmd(); err != nil {
			eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		releaseMain(c.Context())
	},
}

func init() {
	addNodeFlag(releaseCmd, "If present, select drives from given nodes")
	addDriveNameFlag(releaseCmd, "If present, select drives by given names")
	addAccessTierFlag(releaseCmd, "If present, select drives by access-tier")
	addDriveStatusFlag(releaseCmd, "If present, select drives by drive status")
	addAllFlag(releaseCmd, "If present, select all drives")
	addDryRunFlag(releaseCmd)
}

func validateReleaseCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}

	if err := validateDriveNameArgs(); err != nil {
		return err
	}

	if err := validateAccessTierArgs(); err != nil {
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
	case len(nodeArgs) != 0:
	case len(driveNameArgs) != 0:
	case len(accessTierArgs) != 0:
	case len(driveStatusArgs) != 0:
	case len(driveIDArgs) != 0:
	default:
		return errors.New("no drive selected to release")
	}

	if allFlag {
		nodeArgs = nil
		driveNameArgs = nil
		accessTierArgs = nil
		driveStatusSelectors = nil
		driveIDSelectors = nil
	}

	return nil
}

func releaseMain(ctx context.Context) {
	var processed bool
	var failed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := drive.NewLister().
		NodeSelector(toLabelValues(nodeArgs)).
		DriveNameSelector(toLabelValues(driveNameArgs)).
		AccessTierSelector(toLabelValues(accessTierArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		processed = true

		switch result.Drive.Status.Status {
		case directpvtypes.DriveStatusReleased:
			eprintf(quietFlag, false, "%v/%v already released\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
		default:
			volumeCount := result.Drive.GetVolumeCount()
			if volumeCount > 0 {
				failed = true
				eprintf(quietFlag, true, "%v/%v: %v volumes still exist\n", result.Drive.GetNodeID(), result.Drive.GetDriveName(), volumeCount)
			} else {
				result.Drive.Status.Status = directpvtypes.DriveStatusReleased
				var err error
				if !dryRunFlag {
					_, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{})
				}
				if err != nil {
					failed = true
					eprintf(quietFlag, true, "%v/%v: %v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName(), err)
				} else {
					fmt.Printf("Releasing %v/%v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
				}
			}
		}
	}

	if !processed {
		if allFlag {
			eprintf(quietFlag, false, "No resources found\n")
		} else {
			eprintf(quietFlag, false, "No matching resources found\n")
		}

		os.Exit(1)
	}

	if failed {
		os.Exit(1)
	}
}
