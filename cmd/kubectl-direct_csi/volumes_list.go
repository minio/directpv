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
	"fmt"
	"os"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"k8s.io/klog"
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
	dclient := utils.GetDirectCSIClient().DirectCSIDrives()
	vclient := utils.GetDirectCSIClient().DirectCSIVolumes()

	driveList, err := dclient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(driveList.Items) == 0 {
		klog.Errorf("No resource of %s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	accessTierSet, aErr := getAccessTierSet(accessTiers)
	if aErr != nil {
		return aErr
	}
	filterDrives := []directcsi.DirectCSIDrive{}
	for _, d := range driveList.Items {
		if d.MatchGlob(nodes, drives, status) {
			if d.MatchAccessTier(accessTierSet) {
				filterDrives = append(filterDrives, d)
			}
		}
	}

	vols := []directcsi.DirectCSIVolume{}

	drivePaths := map[string]string{}
	driveUUIDs := map[string]string{}
	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		return strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
	}

	for _, d := range filterDrives {
		if d.Status.DriveStatus == directcsi.DriveStatusUnavailable {
			continue
		}
		drivePaths[d.Name] = driveName(d.Status.Path)
		driveUUIDs[d.Name] = d.Status.FilesystemUUID
		for _, f := range d.GetFinalizers() {
			if strings.HasPrefix(f, directcsi.DirectCSIDriveFinalizerPrefix) {
				name := strings.ReplaceAll(f, directcsi.DirectCSIDriveFinalizerPrefix, "")
				vol, err := vclient.Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if vol.MatchStatus(volumeStatus) && vol.MatchPodName(podNames) && vol.MatchPodNamespace(podNss) {
					vols = append(vols, *vol)
				}
			}
		}
	}

	wrappedVolumeList := directcsi.DirectCSIVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: utils.DirectCSIGroupVersion,
		},
		Items: vols,
	}

	if yaml {
		if err := printYAML(wrappedVolumeList); err != nil {
			return err
		}
		return nil
	}
	if json {
		if err := printJSON(wrappedVolumeList); err != nil {
			return err
		}
		return nil
	}

	defaultHeaders := table.Row{
		"VOLUME",
		"CAPACITY",
		"NODE",
		"DRIVE",
		"PODNAME",
		"PODNAMESPACE",
	}

	if wide {
		defaultHeaders = append(defaultHeaders,
			"DRIVEUUID")
	}

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(defaultHeaders)

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, v := range vols {
		emptyOrBytes := func(val int64) string {
			if val == 0 {
				return "-"
			}
			return humanize.IBytes(uint64(val))
		}
		row := []interface{}{
			v.Name,                                //VOLUME
			emptyOrBytes(v.Status.TotalCapacity),  //CAPACITY
			v.Status.NodeName,                     //SERVER
			driveName(drivePaths[v.Status.Drive]), //DRIVE
			printableString(v.ObjectMeta.Labels[directcsi.Group+"/pod.name"]),
			printableString(v.ObjectMeta.Labels[directcsi.Group+"/pod.namespace"]),
		}
		if wide {
			row = append(row, driveUUIDs[v.Status.Drive])
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
