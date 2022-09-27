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

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var (
	lostOnly bool
	showPVC  bool
)

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes served by " + consts.AppPrettyName + ".",
	Example: strings.ReplaceAll(
		`# List all staged and published volumes
$ kubectl {PLUGIN_NAME} volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl {PLUGIN_NAME} volumes ls --node=node1

# Combine multiple filters using csv
$ kubectl {PLUGIN_NAME} vol ls --node=node1,node2 --status=staged --drive=/dev/nvme0n1

# List all published volumes by pod name
$ kubectl {PLUGIN_NAME} volumes ls --status=published --pod-name=minio-{1...3}

# List all published volumes by pod namespace
$ kubectl {PLUGIN_NAME} volumes ls --status=published --pod-namespace=tenant-{1...3}

# List all volumes provisioned based on drive and volume ellipses
$ kubectl {PLUGIN_NAME} volumes ls --drive /dev/xvd{a...d} --node node{1...4}

# List all volumes and its PVC names
$ kubectl {PLUGIN_NAME} volumes ls --all --pvc

# List all the "lost" volumes
$ kubectl {PLUGIN_NAME} volumes ls --lost

# List all the "lost" volumes with their PVC names
$ kubectl {PLUGIN_NAME} volumes ls --lost --pvc`,
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
	listVolumesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter output by drives optionally in ellipses pattern.")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter output by nodes optionally in ellipses pattern.")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&volumeStatusArgs, "status", "s", volumeStatusArgs, fmt.Sprintf("Filter output by volume status. One of %v|%v|%v.", strings.ToLower(string(directpvtypes.VolumeConditionTypePublished)), strings.ToLower(string(directpvtypes.VolumeConditionTypeStaged)), strings.ToLower(string(directpvtypes.VolumeConditionTypeReady))))
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNameArgs, "pod-name", "", podNameArgs, "Filter output by pod names optionally in ellipses pattern.")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNSArgs, "pod-namespace", "", podNSArgs, "Filter output by pod namespaces optionally in ellipses pattern.")
	listVolumesCmd.PersistentFlags().BoolVarP(&lostOnly, "lost", "", lostOnly, "Show lost volumes only.")
	listVolumesCmd.PersistentFlags().BoolVarP(&showPVC, "pvc", "", showPVC, "Show PVC names of the corresponding volumes.")
	listVolumesCmd.PersistentFlags().BoolVarP(&allFlag, "all", "a", allFlag, "List all volumes including not provisioned.")
}

func getPVCName(ctx context.Context, volume types.Volume) string {
	pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
	if err == nil && pv != nil && pv.Spec.ClaimRef != nil {
		return pv.Spec.ClaimRef.Name
	}
	return "-"
}

func listVolumes(ctx context.Context, args []string) error {
	volumeList, err := getFilteredVolumeList(
		ctx,
		func(volume types.Volume) bool {
			if lostOnly {
				return k8s.IsCondition(volume.Status.Conditions,
					string(directpvtypes.VolumeConditionTypeReady),
					metav1.ConditionFalse,
					string(directpvtypes.VolumeConditionReasonDriveLost),
					string(directpvtypes.VolumeConditionMessageDriveLost),
				)
			}
			return allFlag || k8s.IsConditionStatus(volume.Status.Conditions, string(directpvtypes.VolumeConditionTypeReady), metav1.ConditionTrue)
		},
	)
	if err != nil {
		return err
	}

	wrappedVolumeList := types.VolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: string(types.VersionLabelKey),
		},
		Items: volumeList,
	}
	if yamlOutput || jsonOutput {
		if err := printer(wrappedVolumeList); err != nil {
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
		"",
	}
	if wideOutput {
		headers = append(headers, "DRIVENAME")
	}
	if showPVC {
		headers = append(headers, "PVC")
	}

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if !noHeaders {
		t.AppendHeader(headers)
	}

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, volume := range volumeList {
		msg := ""
		for _, c := range volume.Status.Conditions {
			switch c.Type {
			case string(directpvtypes.VolumeConditionTypeReady):
				if c.Status != metav1.ConditionTrue {
					if c.Message != "" {
						msg = color.HiRedString("*" + c.Message)
					}
				}
			}
		}
		row := []interface{}{
			volume.Name, // VOLUME
			printableBytes(volume.Status.TotalCapacity),             // CAPACITY
			volume.Status.NodeName,                                  // SERVER
			getLabelValue(&volume, string(types.DrivePathLabelKey)), // DRIVE
			printableString(volume.Labels[string(types.PodNameLabelKey)]),
			printableString(volume.Labels[string(types.PodNSLabelKey)]),
			msg,
		}
		if wideOutput {
			row = append(row, getLabelValue(&volume, string(types.DriveLabelKey)))
		}
		if showPVC {
			row = append(row, getPVCName(ctx, volume))
		}
		t.AppendRow(row)
	}
	t.SortBy(
		[]table.SortBy{
			{
				Name: "PODNAMESPACE",
				Mode: table.Asc,
			},
		})

	t.Render()
	return nil
}
