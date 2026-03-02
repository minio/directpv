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
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var listDrivesCmd = &cobra.Command{
	Use:           "drives [DRIVE ...]",
	Aliases:       []string{"drive", "dr"},
	Short:         "List drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. List all ready drives
   $ kubectl {PLUGIN_NAME} list drives

2. List all drives from a node
   $ kubectl {PLUGIN_NAME} list drives --nodes=node1

3. List a drive from all nodes
   $ kubectl {PLUGIN_NAME} list drives --drives=nvme1n1

4. List specific drives from specific nodes
   $ kubectl {PLUGIN_NAME} list drives --nodes=node{1...4} --drives=sd{a...f}

5. List drives are in 'error' status
   $ kubectl {PLUGIN_NAME} list drives --status=error

6. List all drives from all nodes with all information.
   $ kubectl {PLUGIN_NAME} list drives --output wide

7. List drives with labels.
   $ kubectl {PLUGIN_NAME} list drives --show-labels

8. List drives filtered by labels
   $ kubectl {PLUGIN_NAME} list drives --labels tier=hot`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args
		if err := validateListDrivesArgs(); err != nil {
			eprintf(true, "%v\n", err)
			os.Exit(-1)
		}

		listDrivesMain(c.Context())
	},
}

func init() {
	setFlagOpts(listDrivesCmd)

	addDriveStatusFlag(listDrivesCmd, "Filter output by drive status")
	addShowLabelsFlag(listDrivesCmd)
	addLabelsFlag(listDrivesCmd, "Filter output by drive labels")
	addAllFlag(listDrivesCmd, "If present, list all drives")
}

func validateListDrivesArgs() error {
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
	case len(labelArgs) != 0:
	default:
		driveStatusSelectors = append(driveStatusSelectors, directpvtypes.DriveStatusReady)
	}

	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveStatusSelectors = nil
		driveIDSelectors = nil
		labelSelectors = nil
	}

	return nil
}

func listDrivesMain(ctx context.Context) {
	drives, err := adminClient.NewDriveLister().
		NodeSelector(directpvtypes.ToLabelValues(nodesArgs)).
		DriveNameSelector(directpvtypes.ToLabelValues(drivesArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		LabelSelector(labelSelectors).
		Get(ctx)
	if err != nil {
		eprintf(true, "%v\n", err)
		os.Exit(1)
	}

	if dryRunPrinter != nil {
		driveList := types.DriveList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			Items: drives,
		}
		dryRunPrinter(driveList)
		return
	}

	headers := table.Row{
		"DRIVE ID",
		"NODE",
		"NAME",
		"MAKE",
		"SIZE",
		"FREE",
		"VOLUMES",
		"STATUS",
	}
	if showLabels {
		headers = append(headers, "LABELS")
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
			volumes = strconv.Itoa(volumeCount)
		}
		status := drive.Status.Status
		if drive.IsUnschedulable() {
			status += ",SchedulingDisabled"
		}
		if drive.IsSuspended() {
			status += ",Suspended"
		}
		driveMake := drive.Status.Make
		if driveMake == "" {
			driveMake = "-"
		}
		row := []interface{}{
			drive.GetDriveID(),
			drive.GetNodeID(),
			drive.GetDriveName(),
			driveMake,
			printableBytes(drive.Status.TotalCapacity),
			printableBytes(drive.Status.FreeCapacity),
			volumes,
			status,
		}
		if showLabels {
			row = append(row, labelsToString(drive.GetLabels()))
		}

		writer.AppendRow(row)
	}

	if writer.Length() != 0 {
		writer.Render()
		return
	}

	if allFlag {
		eprintf(false, "No resources found\n")
	} else {
		eprintf(false, "No matching resources found\n")
	}

	os.Exit(1)
}
