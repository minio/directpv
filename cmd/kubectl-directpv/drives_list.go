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

var drivesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List drives.",
	Example: strings.ReplaceAll(
		`# List all drives
$ kubectl {PLUGIN_NAME} drives ls

# List all drives from a particular node
$ kubectl {PLUGIN_NAME} drives ls --node=node1

# List specified drives from specified nodes
$ kubectl {PLUGIN_NAME} drives ls --node=node1,node2 --drive=/dev/nvme0n1

# List all drives filtered by specified drive ellipsis
$ kubectl {PLUGIN_NAME} drives ls --drive=/dev/sd{a...b}

# List all drives filtered by specified node ellipsis
$ kubectl {PLUGIN_NAME} drives ls --node=node{0...3}

# List all drives by specified combination of node and drive ellipsis
$ kubectl {PLUGIN_NAME} drives ls --drive /dev/xvd{a...d} --node node{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		drivesListMain(c.Context(), args)
	},
}

func drivesListMain(ctx context.Context, args []string) {
	drives, err := drive.GetDriveList(ctx, nodeSelectors, driveSelectors, accessTierSelectors, driveStatusArgs)
	if err != nil {
		eprintf(err.Error(), true)
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
			eprintf(fmt.Sprintf("unable to %v marshal drives; %v", outputFormat, err), true)
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

	if len(driveStatusArgs) == 0 && len(accessTierArgs) == 0 {
		eprintf("No resources found", false)
	} else {
		eprintf("No matching resources found", false)
	}

	os.Exit(1)
}
