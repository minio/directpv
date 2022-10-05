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
	"github.com/jedib0t/go-pretty/v6/text"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var showLostOnly bool

var listDrivesCmd = &cobra.Command{
	Use:   "list",
	Short: "List drives formatted by " + consts.AppPrettyName + ".",
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
	RunE: func(c *cobra.Command, args []string) error {
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		return listDrives(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listDrivesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter by drive path (supports ellipses pattern).")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter by nodes (supports ellipses pattern).")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&accessTierArgs, "access-tier", "", accessTierArgs, fmt.Sprintf("Filter by access tier set on the drive [%s]", strings.Join(directpvtypes.SupportedAccessTierValues(), ", ")))
	listDrivesCmd.PersistentFlags().BoolVarP(&showLostOnly, "lost", "", showLostOnly, "Show only \"Lost\" drives")
}

func listDrives(ctx context.Context, args []string) error {
	resultCh, err := drive.ListDrives(
		ctx,
		nodeSelectors,
		driveSelectors,
		accessTierSelectors,
		k8s.MaxThreadCount,
	)
	if err != nil {
		return err
	}

	drives := []types.Drive{}
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}
		if showLostOnly && !result.Drive.IsLost() {
			continue
		}
		drives = append(drives, result.Drive)
	}

	if yamlOutput || jsonOutput {
		driveList := types.DriveList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: string(types.VersionLabelKey),
			},
			Items: drives,
		}
		if err := printer(driveList); err != nil {
			klog.ErrorS(err, "unable to marshal drives", "format", outputFormat)
			return err
		}
		return nil
	}

	headers := table.Row{
		"PATH",
		"SIZE",
		"ALLOCATED",
		"VOLUMES",
		"STATUS",
		"NODE",
		"ACCESSTIER",
	}
	if wideOutput {
		headers = append(headers, "MODEL")
		headers = append(headers, "VENDOR")
	}

	text.DisableColors()
	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	if !noHeaders {
		writer.AppendHeader(headers)
	}

	style := table.StyleColoredDark
	style.Options = table.OptionsDefault
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	writer.SetStyle(style)

	for _, drive := range drives {
		volumes := "-"
		if len(drive.Finalizers) > 1 {
			volumes = fmt.Sprintf("%v", len(drive.Finalizers)-1)
		}
		row := []interface{}{
			drive.Status.Path,
			printableBytes(drive.Status.TotalCapacity),
			func() string {
				// Show allocated capacity only if the drive has volumes scheduled in it
				if volumes != "-" {
					return printableBytes(drive.Status.AllocatedCapacity)
				}
				return "-"
			}(),
			volumes,
			drive.Status.Status,
			drive.Status.NodeName,
			func() string {
				if drive.Status.AccessTier == directpvtypes.AccessTierUnknown {
					return "-"
				}
				return string(drive.Status.AccessTier)
			}(),
		}
		if wideOutput {
			row = append(row, drive.Status.ModelNumber)
			row = append(row, drive.Status.Vendor)
		}
		writer.AppendRow(row)
	}
	writer.SortBy(
		[]table.SortBy{
			{
				Name: "PATH",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "SIZE",
				Mode: table.Asc,
			},
			{
				Name: "ALLOCATED",
				Mode: table.Asc,
			},
		},
	)
	writer.Render()
	return nil
}
