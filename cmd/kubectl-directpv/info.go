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

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var infoCmd = &cobra.Command{
	Use:           "info",
	Short:         "Show information about " + consts.AppPrettyName + " installation",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(c *cobra.Command, _ []string) {
		infoMain(c.Context())
	},
}

func infoMain(ctx context.Context) {
	crds, err := k8s.CRDClient().List(ctx, metav1.ListOptions{})
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to list CRDs; %v\n", err)
		os.Exit(1)
	}

	drivesFound := false
	volumesFound := false
	for _, crd := range crds.Items {
		if strings.Contains(crd.Name, consts.DriveResource+"."+consts.GroupName) {
			drivesFound = true
		}
		if strings.Contains(crd.Name, consts.VolumeResource+"."+consts.GroupName) {
			volumesFound = true
		}
	}
	if !drivesFound || !volumesFound {
		utils.Eprintf(quietFlag, false, "%v installation not found\n", consts.AppPrettyName)
		os.Exit(1)
	}

	nodeList, err := k8s.GetCSINodes(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if len(nodeList) == 0 {
		utils.Eprintf(quietFlag, true, "%v not installed\n", consts.AppPrettyName)
		os.Exit(1)
	}

	drives, err := client.NewDriveLister().Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to get drive list; %v\n", err)
		os.Exit(1)
	}

	volumes, err := client.NewVolumeLister().Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to get volume list; %v\n", err)
		os.Exit(1)
	}

	writer := newTableWriter(
		table.Row{"NODE", "CAPACITY", "ALLOCATED", "VOLUMES", "DRIVES"},
		[]table.SortBy{
			{
				Name: "DRIVES",
				Mode: table.Asc,
			},
			{
				Name: "VOLUMES",
				Mode: table.Asc,
			},
			{
				Name: "ALLOCATED",
				Mode: table.Asc,
			},
			{
				Name: "CAPACITY",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
				Mode: table.Asc,
			},
		},
		false,
	)

	var totalDriveSize uint64
	var totalVolumeSize uint64
	for _, n := range nodeList {
		driveCount := 0
		driveSize := uint64(0)
		for _, d := range drives {
			if string(d.GetNodeID()) == n {
				driveCount++
				driveSize += uint64(d.Status.TotalCapacity)
			}
		}
		totalDriveSize += driveSize

		volumeCount := 0
		volumeSize := uint64(0)
		for _, v := range volumes {
			if string(v.GetNodeID()) == n {
				if v.IsPublished() {
					volumeCount++
					volumeSize += uint64(v.Status.TotalCapacity)
				}
			}
		}
		totalVolumeSize += volumeSize

		if driveCount == 0 {
			writer.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", color.HiYellowString(dot), n),
				"-",
				"-",
				"-",
				"-",
			})
		} else {
			writer.AppendRow([]interface{}{
				fmt.Sprintf("%s %s", color.GreenString(dot), n),
				humanize.IBytes(driveSize),
				humanize.IBytes(volumeSize),
				fmt.Sprintf("%d", volumeCount),
				fmt.Sprintf("%d", driveCount),
			})
		}
	}

	if !quietFlag {
		writer.Render()
		if len(drives) > 0 {
			fmt.Printf(
				"\n%s/%s used, %s volumes, %s drives\n",
				humanize.IBytes(totalVolumeSize),
				humanize.IBytes(totalDriveSize),
				color.HiWhiteString("%d", len(volumes)),
				color.HiWhiteString("%d", len(drives)),
			)
		}
	}
}
