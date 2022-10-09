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
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	showPVC    bool
	stagedFlag bool
)

var volumesListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List volumes.",
	Example: strings.ReplaceAll(
		`# List all published volumes
$ kubectl {PLUGIN_NAME} volumes ls

# List all published volumes from a particular node
$ kubectl {PLUGIN_NAME} volumes ls --node=node1

# List all staged volumes from specified nodes on specified drive
$ kubectl {PLUGIN_NAME} vol ls --node=node1,node2 --staged --drive=/dev/nvme0n1

# List all published volumes by pod name
$ kubectl {PLUGIN_NAME} volumes ls --pod-name=minio-{1...3}

# List all published volumes by pod namespace
$ kubectl {PLUGIN_NAME} volumes ls --pod-namespace=tenant-{1...3}

# List all published volumes from range of nodes and drives
$ kubectl {PLUGIN_NAME} volumes ls --drive /dev/xvd{a...d} --node node{1...4}

# List all volumes including PVC names
$ kubectl {PLUGIN_NAME} volumes ls --all --pvc`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumesListMain(c.Context(), args)
	},
}

func init() {
	volumesListCmd.PersistentFlags().BoolVarP(&stagedFlag, "staged", "", stagedFlag, "Show only volumes in staged state.")
	volumesListCmd.PersistentFlags().BoolVarP(&showPVC, "pvc", "", showPVC, "Show PVC names of the corresponding volumes.")
}

func getPVCName(ctx context.Context, volume types.Volume) string {
	pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
	if err == nil && pv != nil && pv.Spec.ClaimRef != nil {
		return pv.Spec.ClaimRef.Name
	}
	return "-"
}

func volumesListMain(ctx context.Context, args []string) {
	resultCh, err := volume.ListVolumes(
		ctx,
		nodeSelectors,
		driveSelectors,
		podNameSelectors,
		podNSSelectors,
		k8s.MaxThreadCount,
	)
	if err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}

	volumes := []types.Volume{}
	for result := range resultCh {
		if result.Err != nil {
			eprintf(result.Err.Error(), true)
			os.Exit(1)
		}

		if allFlag || (stagedFlag && result.Volume.IsStaged()) || result.Volume.IsPublished() {
			volumes = append(volumes, result.Volume)
		}
	}

	if yamlOutput || jsonOutput {
		volumeList := types.VolumeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: string(directpvtypes.VersionLabelKey),
			},
			Items: volumes,
		}
		if err := printer(volumeList); err != nil {
			eprintf(fmt.Sprintf("unable to %v marshal volumes; %v", outputFormat, err), true)
			os.Exit(1)
		}

		return
	}

	headers := table.Row{
		"VOLUME",
		"CAPACITY",
		"NODE",
		"DRIVE",
		"PODNAME",
		"PODNAMESPACE",
		"STATUS",
	}
	if wideOutput {
		headers = append(headers, "DRIVE ID")
	}
	if showPVC {
		headers = append(headers, "PVC")
	}
	writer := newTableWriter(
		headers,
		[]table.SortBy{
			{
				Name: "PODNAMESPACE",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
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
				Name: "DRIVE",
				Mode: table.Asc,
			},
			{
				Name: "PODNAME",
				Mode: table.Asc,
			},
			{
				Name: "VOLUME",
				Mode: table.Asc,
			},
		},
		noHeaders)

	for _, volume := range volumes {
		status := string(volume.GetStatus())
		switch {
		case volume.IsReleased():
			status = "Released"
		case volume.IsDriveLost():
			status = "Lost"
		case volume.IsPublished():
			status = "Bounded"
		}

		row := []interface{}{
			volume.Name,
			printableBytes(volume.Status.TotalCapacity),
			volume.GetNodeID(),
			printableString(string(volume.GetDriveName())),
			printableString(volume.GetPodName()),
			printableString(volume.GetPodNS()),
			status,
		}
		if wideOutput {
			row = append(row, volume.GetDriveID())
		}
		if showPVC {
			row = append(row, getPVCName(ctx, volume))
		}

		writer.AppendRow(row)
	}

	if writer.Length() > 0 {
		writer.Render()
		return
	}

	if len(nodeSelectors) != 0 || len(driveSelectors) != 0 || len(podNameSelectors) != 0 || len(podNSSelectors) != 0 || stagedFlag {
		eprintf("No matching resources found", false)
	} else {
		eprintf("No resources found", false)
	}

	os.Exit(1)
}
