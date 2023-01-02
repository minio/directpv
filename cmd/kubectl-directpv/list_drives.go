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
	"github.com/minio/directpv/pkg/utils"
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
		`# List all ready drives
$ kubectl {PLUGIN_NAME} list drives

# List all drives from a node
$ kubectl {PLUGIN_NAME} list drives --nodes=node1

# List a drive from all nodes
$ kubectl {PLUGIN_NAME} list drives --drives=nvme1n1

# List specific drives from specific nodes
$ kubectl {PLUGIN_NAME} list drives --nodes=node{1...4} --drives=sd{a...f}

# List drives are in 'error' status
$ kubectl {PLUGIN_NAME} list drives --status=error

# List all drives from all nodes with all information.
$ kubectl {PLUGIN_NAME} list drives --output wide

# List drives with labels.
$ kubectl {PLUGIN_NAME} list drives --show-labels`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		driveIDArgs = args
		if err := validateListDrivesArgs(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		listDrivesMain(c.Context(), args)
	},
}

func init() {
	listDrivesCmd.Flags().SortFlags = false
	listDrivesCmd.InheritedFlags().SortFlags = false
	listDrivesCmd.LocalFlags().SortFlags = false
	listDrivesCmd.LocalNonPersistentFlags().SortFlags = false
	listDrivesCmd.NonInheritedFlags().SortFlags = false
	listDrivesCmd.PersistentFlags().SortFlags = false

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

func listDrivesMain(ctx context.Context, args []string) {
	drives, err := drive.NewLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		StatusSelector(driveStatusSelectors).
		DriveIDSelector(driveIDSelectors).
		LabelSelector(labelSelectors).
		Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
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
			utils.Eprintf(quietFlag, true, "unable to %v marshal drives; %v\n", outputFormat, err)
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
	}
	if wideOutput {
		headers = append(headers, "DRIVE ID")
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
		}
		if wideOutput {
			row = append(row, drive.GetDriveID())
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
		utils.Eprintf(quietFlag, false, "No resources found\n")
	} else {
		utils.Eprintf(quietFlag, false, "No matching resources found\n")
	}

	os.Exit(1)
}
