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
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var volumeNameArgs []string

var getVolumesCmd = &cobra.Command{
	Use:     "volumes [VOLUME ...]",
	Aliases: []string{"volume", "vol"},
	Short:   "List volumes.",
	Example: strings.ReplaceAll(
		`# Get all ready volumes
$ kubectl {PLUGIN_NAME} get volumes

# Get all volumes from all nodes with all information include PVC name.
$ kubectl {PLUGIN_NAME} get drives --all --pvc --output wide

# Get volumes in Pending state
$ kubectl {PLUGIN_NAME} get volumes --status=pending

# Get volumes served by a node
$ kubectl {PLUGIN_NAME} get volumes --node=node1

# Get volumes served by a drive ID
$ kubectl {PLUGIN_NAME} get volumes --drive=b84758b0-866f-4a12-9d00-d8f7da76ceb3

# Get volumes served by drives on nodes
$ kubectl {PLUGIN_NAME} get volumes --node=node1,node2 --drive=nvme0n1

# Get volumes by pod name
$ kubectl {PLUGIN_NAME} get volumes --pod-name=minio-{1...3}

# Get volumes by pod namespace
$ kubectl {PLUGIN_NAME} get volumes --pod-namespace=tenant-{1...3}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args

		if err := validateGetVolumesCmd(); err != nil {
			eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		getVolumesMain(c.Context(), args)
	},
}

func init() {
	addDriveIDFlag(getVolumesCmd, "Filter output by drive IDs")
	addPodNameFlag(getVolumesCmd, "Filter output by pod names")
	addPodNSFlag(getVolumesCmd, "Filter output by pod namespaces")
	getVolumesCmd.PersistentFlags().BoolVar(&pvcFlag, "pvc", pvcFlag, "Add PVC names in the output")
	addVolumeStatusFlag(getVolumesCmd, "Filter output by volume status")
}

func validateGetVolumesCmd() error {
	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	if err := validatePodNameArgs(); err != nil {
		return err
	}

	if err := validatePodNSArgs(); err != nil {
		return err
	}

	if err := validateVolumeNameArgs(); err != nil {
		return err
	}

	if err := validateVolumeStatusArgs(); err != nil {
		return err
	}

	switch {
	case allFlag:
	case len(nodeArgs) != 0:
	case len(driveNameArgs) != 0:
	case len(driveIDArgs) != 0:
	case len(podNameArgs) != 0:
	case len(podNSArgs) != 0:
	case len(volumeNameArgs) != 0:
	case len(volumeStatusArgs) != 0:
	default:
		volumeStatusSelectors = append(volumeStatusSelectors, directpvtypes.VolumeStatusReady)
	}

	if allFlag {
		nodeArgs = nil
		driveNameArgs = nil
		driveIDSelectors = nil
		podNameArgs = nil
		podNSArgs = nil
		volumeNameArgs = nil
		volumeStatusSelectors = nil
	}

	return nil
}

func getPVCName(ctx context.Context, volume types.Volume) string {
	pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
	if err == nil && pv != nil && pv.Spec.ClaimRef != nil {
		return pv.Spec.ClaimRef.Name
	}
	return "-"
}

func getVolumesMain(ctx context.Context, args []string) {
	volumes, err := volume.NewLister().
		NodeSelector(toLabelValues(nodeArgs)).
		DriveNameSelector(toLabelValues(driveNameArgs)).
		DriveIDSelector(toLabelValues(driveIDArgs)).
		PodNameSelector(toLabelValues(podNameArgs)).
		PodNSSelector(toLabelValues(podNSArgs)).
		StatusSelector(volumeStatusSelectors).
		VolumeNameSelector(volumeNameArgs).
		Get(ctx)
	if err != nil {
		eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
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
			eprintf(quietFlag, true, "unable to %v marshal volumes; %v\n", outputFormat, err)
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
	if pvcFlag {
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
		status := string(volume.Status.Status)
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
		if pvcFlag {
			row = append(row, getPVCName(ctx, volume))
		}

		writer.AppendRow(row)
	}

	if writer.Length() > 0 {
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
