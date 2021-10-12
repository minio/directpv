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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var podNames, podNss []string

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "list volumes in the DirectCSI cluster",
	Long:  "",
	Example: `

# List all staged and published volumes
$ kubectl direct-csi volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl direct-csi volumes ls --nodes=directcsi-1

# Combine multiple filters using csv
$ kubectl direct-csi vol ls --nodes=directcsi-1,directcsi-2 --status=staged --drives=/dev/nvme0n1

# List all published volumes by pod name
$ kubectl direct-csi volumes ls --status=published --pod-name=minio-{1...3}

# List all published volumes by pod namespace
$ kubectl direct-csi volumes ls --status=published --pod-namespace=tenant-{1...3}

# List all volumes provisioned based on drive and volume ellipses
$ kubectl direct-csi volumes ls --drives '/dev/xvd{a...d} --nodes 'node-{1...4}''

`,
	RunE: func(c *cobra.Command, args []string) error {
		return listVolumes(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listVolumesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "ellipses expander for drive paths")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "ellipses expander for node names")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&volumeStatus, "status", "s", volumeStatus, "match based on volume status. The possible values are [staged,published]")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNames, "pod-name", "", podNames, "ellipses expander for pod names")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&podNss, "pod-namespace", "", podNss, "ellipses expander for pod namespace")
	listVolumesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "list all volumes (including non-provisioned)")
}

func listVolumes(ctx context.Context, args []string) error {

	volumeList, err := getFilteredVolumeList(
		ctx,
		utils.GetDirectCSIClient().DirectCSIVolumes(),
		func(volume directcsi.DirectCSIVolume) bool {
			if all {
				return true
			}
			return utils.IsConditionStatus(volume.Status.Conditions, string(directcsi.DirectCSIVolumeConditionReady), metav1.ConditionTrue) && volume.MatchStatus(volumeStatus)
		},
	)
	if err != nil {
		return err
	}

	wrappedVolumeList := directcsi.DirectCSIVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: utils.DirectCSIGroupVersion,
		},
		Items: volumeList,
	}
	if yaml || json {
		if err := printer(wrappedVolumeList); err != nil {
			klog.ErrorS(err, "error marshaling volumes", "format", outputMode)
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
	if wide {
		headers = append(headers, "DRIVENAME")
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
	for _, volume := range volumeList {
		row := []interface{}{
			volume.Name, //VOLUME
			emptyOrBytes(volume.Status.TotalCapacity),                         //CAPACITY
			volume.Status.NodeName,                                            //SERVER
			driveName(utils.GetLabelV(&volume, utils.ReservedDrivePathLabel)), //DRIVE
			printableString(volume.Labels[directcsi.Group+"/pod.name"]),
			printableString(volume.Labels[directcsi.Group+"/pod.namespace"]),
		}
		if wide {
			row = append(row, utils.GetLabelV(&volume, utils.DriveLabel))
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
