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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var volumeNameArgs []string

var listVolumesCmd = &cobra.Command{
	Use:           "volumes [VOLUME ...]",
	Aliases:       []string{"volume", "vol"},
	Short:         "List volumes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. List all ready volumes
   $ kubectl {PLUGIN_NAME} list volumes

2. List volumes served by a node
   $ kubectl {PLUGIN_NAME} list volumes --nodes=node1

3. List volumes served by drives on nodes
   $ kubectl {PLUGIN_NAME} list volumes --nodes=node1,node2 --drives=nvme0n1

4. List volumes by pod name
   $ kubectl {PLUGIN_NAME} list volumes --pod-names=minio-{1...3}

5. List volumes by pod namespace
   $ kubectl {PLUGIN_NAME} list volumes --pod-namespaces=tenant-{1...3}

6. List all volumes from all nodes with all information include PVC name.
   $ kubectl {PLUGIN_NAME} list drives --all --pvc --output wide

7. List volumes in Pending state
   $ kubectl {PLUGIN_NAME} list volumes --status=pending

8. List volumes served by a drive ID
   $ kubectl {PLUGIN_NAME} list volumes --drive-id=b84758b0-866f-4a12-9d00-d8f7da76ceb3

9. List volumes with labels.
   $ kubectl {PLUGIN_NAME} list volumes --show-labels

10. List volumes filtered by labels
   $ kubectl {PLUGIN_NAME} list volumes --labels tier=hot`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args
		if err := validateListVolumesArgs(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		listVolumesMain(c.Context())
	},
}

func init() {
	setFlagOpts(listVolumesCmd)

	addDriveIDFlag(listVolumesCmd, "Filter output by drive IDs")
	addPodNameFlag(listVolumesCmd, "Filter output by pod names")
	addPodNSFlag(listVolumesCmd, "Filter output by pod namespaces")
	listVolumesCmd.PersistentFlags().BoolVar(&pvcFlag, "pvc", pvcFlag, "Add PVC names in the output")
	addVolumeStatusFlag(listVolumesCmd, "Filter output by volume status")
	addShowLabelsFlag(listVolumesCmd)
	addLabelsFlag(listVolumesCmd, "Filter output by volume labels")
	addAllFlag(listVolumesCmd, "If present, list all volumes")
}

func validateListVolumesArgs() error {
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
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(driveIDArgs) != 0:
	case len(podNameArgs) != 0:
	case len(podNSArgs) != 0:
	case len(volumeNameArgs) != 0:
	case len(volumeStatusArgs) != 0:
	case len(labelArgs) != 0:
	default:
		volumeStatusSelectors = append(volumeStatusSelectors, directpvtypes.VolumeStatusReady)
	}

	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveIDSelectors = nil
		podNameArgs = nil
		podNSArgs = nil
		volumeNameArgs = nil
		volumeStatusSelectors = nil
		labelSelectors = nil
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

func listVolumesMain(ctx context.Context) {
	volumes, err := client.NewVolumeLister().
		NodeSelector(utils.ToLabelValues(nodesArgs)).
		DriveNameSelector(utils.ToLabelValues(drivesArgs)).
		DriveIDSelector(utils.ToLabelValues(driveIDArgs)).
		PodNameSelector(utils.ToLabelValues(podNameArgs)).
		PodNSSelector(utils.ToLabelValues(podNSArgs)).
		StatusSelector(volumeStatusSelectors).
		VolumeNameSelector(volumeNameArgs).
		LabelSelector(labelSelectors).
		Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if dryRunPrinter != nil {
		volumeList := types.VolumeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: "v1",
			},
			Items: volumes,
		}
		dryRunPrinter(volumeList)
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
	if showLabels {
		headers = append(headers, "LABELS")
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

		if volume.IsSuspended() {
			status += ",Suspended"
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
		if showLabels {
			row = append(row, labelsToString(volume.GetLabels()))
		}
		writer.AppendRow(row)
	}

	if writer.Length() > 0 {
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
