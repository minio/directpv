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
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var showPVC bool

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes created by " + consts.AppPrettyName + ".",
	Example: strings.ReplaceAll(
		`# List all published volumes
$ kubectl {PLUGIN_NAME} volumes ls

# List all published volumes from a particular node
$ kubectl {PLUGIN_NAME} volumes ls --node=node1

# List all staged volumes provisioned on specified drives from specified nodes
$ kubectl {PLUGIN_NAME} vol ls --node=node1,node2 --staged --drive=/dev/nvme0n1

# Combine multiple filters using csv
$ kubectl {PLUGIN_NAME} volumes list --node=node1,node2 --drive=/dev/nvme0n1

# List all published volumes filtered by specified pod-name
$ kubectl {PLUGIN_NAME} volumes ls --pod-name=minio-1

# List all published volumes filtered by specified pod-name ellipses
$ kubectl {PLUGIN_NAME} volumes ls --pod-name=minio-{1...3}

# List all published volumes filtered by specified pod namespace
$ kubectl {PLUGIN_NAME} volumes ls --pod-namespace=tenant-1

# List all published volumes filtered by specified pod namespace ellipses
$ kubectl {PLUGIN_NAME} volumes ls --pod-namespace=tenant-{1...3}

# List all published volumes by specified combination of node and drive ellipses
$ kubectl {PLUGIN_NAME} volumes ls --drive /dev/xvd{a...d} --node node{1...4}

# List all volumes including PVC names
$ kubectl {PLUGIN_NAME} volumes ls --all --pvc`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	RunE: func(c *cobra.Command, args []string) error {
		if err := validateVolumeSelectors(); err != nil {
			return err
		}
		return listVolumes(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listVolumesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter by drive paths (supports ellipses pattern).")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter by nodes (supports ellipses pattern).")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNameArgs, "pod-name", "", podNameArgs, "Filter by pod names (supports ellipses pattern).")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNSArgs, "pod-namespace", "", podNSArgs, "Filter by pod namespaces (supports ellipses pattern).")
	listVolumesCmd.PersistentFlags().BoolVarP(&stagedFlag, "staged", "", stagedFlag, "Show only volumes in staged state.")
	listVolumesCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "List all volumes including non-provisioned.")
	listVolumesCmd.PersistentFlags().BoolVarP(&showPVC, "pvc", "", showPVC, "Show PVC names of the corresponding volumes.")
}

func getPVCName(ctx context.Context, volume types.Volume) string {
	pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
	if err == nil && pv != nil && pv.Spec.ClaimRef != nil {
		return pv.Spec.ClaimRef.Name
	}
	return "-"
}

func listVolumes(ctx context.Context, args []string) error {
	resultCh, err := volume.ListVolumes(
		ctx,
		nodeSelectors,
		driveSelectors,
		podNameSelectors,
		podNSSelectors,
		k8s.MaxThreadCount,
	)
	if err != nil {
		return err
	}

	volumes := []types.Volume{}
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}

		if allFlag || (stagedFlag && result.Volume.Status.IsStaged()) || result.Volume.Status.IsPublished() {
			volumes = append(volumes, result.Volume)
		}
	}

	if yamlOutput || jsonOutput {
		volumeList := types.VolumeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: string(types.VersionLabelKey),
			},
			Items: volumes,
		}
		if err := printer(volumeList); err != nil {
			klog.ErrorS(err, "unable to marshal volumes", "format", outputFormat)
			return err
		}
		return nil
	}

	headers := table.Row{
		"VOLUME",
		"CAPACITY",
		"NODE",
		"DRIVE",
		"PODNAME",
		"PODNAMESPACE",
	}
	if wideOutput {
		headers = append(headers, "DRIVENAME")
	}
	if showPVC {
		headers = append(headers, "PVC")
	}
	headers = append(headers, "STATUS")

	text.DisableColors()
	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	if !noHeaders {
		writer.AppendHeader(headers)
	}

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	writer.SetStyle(style)

	for _, volume := range volumes {
		row := []interface{}{
			volume.Name,
			printableBytes(volume.Status.TotalCapacity),
			volume.Status.NodeName,
			getLabelValue(&volume, string(types.DrivePathLabelKey)),
			printableString(volume.Labels[string(types.PodNameLabelKey)]),
			printableString(volume.Labels[string(types.PodNSLabelKey)]),
		}
		if wideOutput {
			row = append(row, getLabelValue(&volume, string(types.DriveLabelKey)))
		}
		if showPVC {
			row = append(row, getPVCName(ctx, volume))
		}

		status := "-"
		switch {
		case volume.Status.IsDriveLost():
			status = "Lost"
		case volume.Status.IsPublished():
			status = "Published"
		case volume.Status.IsStaged():
			status = "Staged"
		}
		row = append(row, status)

		writer.AppendRow(row)
	}
	writer.SortBy(
		[]table.SortBy{
			{
				Name: "PODNAMESPACE",
				Mode: table.Asc,
			},
			{
				Name: "VOLUME",
				Mode: table.Asc,
			},
			{
				Name: "STATUS",
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
			{
				Name: "DRIVE",
				Mode: table.Asc,
			},
			{
				Name: "PODNAME",
				Mode: table.Asc,
			},
		},
	)

	writer.Render()
	return nil
}
