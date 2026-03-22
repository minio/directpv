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
	"strconv"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
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
	nodeInfoMap, err := adminClient.Info(ctx)
	if err != nil {
		eprintf(true, "%v\n", err)
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
	var totalDriveCount int
	var totalVolumeCount int
	for n, info := range nodeInfoMap {
		totalDriveSize += info.DriveSize
		totalVolumeSize += info.VolumeSize
		totalDriveCount += info.DriveCount
		totalVolumeCount += info.VolumeCount
		if info.DriveCount == 0 {
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
				utils.IBytes(info.DriveSize),
				utils.IBytes(info.VolumeSize),
				strconv.Itoa(info.VolumeCount),
				strconv.Itoa(info.DriveCount),
			})
		}
	}

	if !quietFlag {
		writer.Render()
		if len(nodeInfoMap) > 0 {
			fmt.Printf(
				"\n%s/%s used, %s volumes, %s drives\n",
				utils.IBytes(totalVolumeSize),
				utils.IBytes(totalDriveSize),
				color.HiWhiteString("%d", totalVolumeCount),
				color.HiWhiteString("%d", totalDriveCount),
			)
		}
	}
}
