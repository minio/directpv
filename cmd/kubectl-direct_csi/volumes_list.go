// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var (
	podNames = []string{}
	podNss   = []string{}
)

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "list volumes in the DirectCSI cluster",
	Long:  "",
	Example: `
# List all volumes provisioned on nvme drives across all nodes 
$ kubectl direct-csi volumes ls --drives '/dev/nvme*'

# List all staged and published volumes
$ kubectl direct-csi volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl direct-csi volumes ls --nodes=directcsi-1

# Combine multiple filters using csv
$ kubectl direct-csi vol ls --nodes=directcsi-1,directcsi-2 --status=staged --drives=/dev/nvme0n1

# List all published volumes by pod name
$ kubectl direct-csi volumes ls --status=published --pod-name=my-minio*

# List all published volumes by pod namespace
$ kubectl direct-csi volumes ls --status=published --pod-namespace=my-minio-ns*
`,
	RunE: func(c *cobra.Command, args []string) error {
		return listVolumes(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listVolumesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glob prefix match for drive paths")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob prefix match for node names")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&volumeStatus, "status", "s", volumeStatus, "filters based on volume status. The possible values are [staged,published]")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "filter based on access-tier")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNames, "pod-name", "", podNames, "glob prefix match for pod names")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNss, "pod-namespace", "", podNss, "glob prefix match for pod namespace")
}

func listVolumes(ctx context.Context, args []string) error {
	accessTierSet, err := getAccessTierSet(accessTiers)
	if err != nil {
		return err
	}

	drives, err := getFilteredDriveList(
		ctx,
		utils.GetDirectCSIClient().DirectCSIDrives(),
		func(drive directcsi.DirectCSIDrive) bool {
			return drive.MatchGlob(nodes, drives, status) && drive.MatchAccessTier(accessTierSet) && drive.Status.DriveStatus != directcsi.DriveStatusUnavailable
		},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	driveMap := map[string]*directcsi.DirectCSIDrive{}
	for i := range drives {
		driveMap[drives[i].Name] = &drives[i]
	}

	volumes, err := utils.GetVolumeList(ctx, utils.GetDirectCSIClient().DirectCSIVolumes(), nil, nil, nil, nil)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("error getting volume list: %v", err)
		return err
	}

	volumeList := directcsi.DirectCSIVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: utils.DirectCSIGroupVersion,
		},
	}
	for _, volume := range volumes {
		if volume.MatchStatus(volumeStatus) && volume.MatchPodName(podNames) && volume.MatchPodNamespace(podNss) {
			if _, found := driveMap[volume.Status.Drive]; found {
				if utils.IsConditionStatus(volume.Status.Conditions, string(directcsi.DirectCSIVolumeConditionReady), metav1.ConditionTrue) {
					volumeList.Items = append(volumeList.Items, volume)
				}
			}
		}
	}

	if yaml {
		return printYAML(volumeList)
	}
	if json {
		return printJSON(volumeList)
	}

	headers := table.Row{
		"VOLUME",
		"CAPACITY",
		"NODE",
		"DRIVE",
		"PODNAME",
		"PODNAMESPACE",
	}
	if wide {
		headers = append(headers, "DRIVEUUID")
	}

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(headers)

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		return strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
	}
	emptyOrBytes := func(val int64) string {
		if val == 0 {
			return "-"
		}
		return humanize.IBytes(uint64(val))
	}
	for _, volume := range volumeList.Items {
		drive := driveMap[volume.Status.Drive]
		row := []interface{}{
			volume.Name, //VOLUME
			emptyOrBytes(volume.Status.TotalCapacity), //CAPACITY
			volume.Status.NodeName,                    //SERVER
			driveName(drive.Status.Path),              //DRIVE
			printableString(volume.Labels[directcsi.Group+"/pod.name"]),
			printableString(volume.Labels[directcsi.Group+"/pod.namespace"]),
		}
		if wide {
			row = append(row, drive.Status.FilesystemUUID)
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
