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

	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var getDrivesCmd = &cobra.Command{
	Use:     "drives [DRIVE ...]",
	Aliases: []string{"drive", "dr"},
	Short:   "Get drives.",
	Example: strings.ReplaceAll(
		`# Get all ready drives
$ kubectl {PLUGIN_NAME} get drives

# Get all drives from all nodes with all information.
$ kubectl {PLUGIN_NAME} get drives --all --output wide

# Get all drives from a node
$ kubectl {PLUGIN_NAME} get drives --node=node1

# Get drives from all nodes
$ kubectl {PLUGIN_NAME} get drives --drive-name=sda

# Get specific drives from specific nodes
$ kubectl {PLUGIN_NAME} get drives --node=node{1...4} --drive-name=sd{a...f}

# Get drives are in 'warm' access-tier
$ kubectl {PLUGIN_NAME} get drives --access-tier=warm

# Get drives are in 'error' status
$ kubectl {PLUGIN_NAME} get drives --status=error`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args

		if err := validateGetDrivesCmd(); err != nil {
			eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		getDrivesMain(c.Context(), args)
	},
}

func init() {
	addAccessTierFlag(getDrivesCmd, "Filter output by access tier")
	addDriveStatusFlag(getDrivesCmd, "Filter output by drive status")
}

func validateGetDrivesCmd() error {
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
		driveStatusSelectors = append(driveStatusSelectors, directpvtypes.DriveStatusReady)
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

func getDrivesMain(ctx context.Context, args []string) {
	drives, err := drive.NewLister().
		NodeSelector(toLabelValues(nodeArgs)).
		DriveNameSelector(toLabelValues(driveNameArgs)).
		AccessTierSelector(toLabelValues(accessTierArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		Get(ctx)
	if err != nil {
		eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if yamlOutput || jsonOutput {
		driveList := types.DriveList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: string(directpvtypes.VersionLabelKey),
			},
			Items: drives,
		}
		if err := printer(driveList); err != nil {
			eprintf(quietFlag, true, "unable to %v marshal drives; %v\n", outputFormat, err)
			os.Exit(1)
		}
		return
	}

	headers := table.Row{
		"NODE",
		"NAME",
		"MAKE",
		"SIZE",
		"FREE",
		"VOLUMES",
		"STATUS",
		"ACCESSTIER",
	}
	if wideOutput {
		headers = append(headers, "DRIVE ID")
	}
	writer := newTableWriter(
		headers,
		[]table.SortBy{
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "STATUS",
				Mode: table.Asc,
			},
			{
				Name: "ACCESSTIER",
				Mode: table.Asc,
			},
			{
				Name: "SIZE",
				Mode: table.Asc,
			},
			{
				Name: "FREE",
				Mode: table.Asc,
			},
			{
				Name: "NAME",
				Mode: table.Asc,
			},
		},
		noHeaders,
	)

	for _, drive := range drives {
		volumeCount := drive.GetVolumeCount()
		volumes := "-"
		if volumeCount > 0 {
			volumes = fmt.Sprintf("%v", volumeCount)
		}
		status := drive.Status.Status
		if drive.IsUnschedulable() {
			status += ",SchedulingDisabled"
		}
		row := []interface{}{
			drive.GetNodeID(),
			drive.GetDriveName(),
			drive.Status.Make,
			printableBytes(drive.Status.TotalCapacity),
			printableBytes(drive.Status.FreeCapacity),
			volumes,
			status,
			func() string {
				if drive.GetAccessTier() == directpvtypes.AccessTierDefault {
					return "-"
				}
				return string(drive.GetAccessTier())
			}(),
		}
		if wideOutput {
			row = append(row, drive.GetDriveID())
		}

		writer.AppendRow(row)
	}

	if writer.Length() != 0 {
		writer.Render()
		return
	}

	if allFlag {
		eprintf(quietFlag, false, "No resources found\n")
	} else {
		eprintf(quietFlag, false, "No matching resources found\n")
	}

	os.Exit(1)
}
