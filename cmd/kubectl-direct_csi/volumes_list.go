// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"path/filepath"
	"strings"

	directv1beta1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mb0/glob"
	"github.com/spf13/cobra"
)

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "list volumes in the DirectCSI cluster",
	Long:  "",
	Example: `
# List all volumes provisioned on nvme drives across all nodes 
$ kubectl direct-csi volumes ls --drives=/dev/nvme*

# List all staged and published volumes
$ kubectl direct-csi volumes ls --status=staged,published

# List all volumes from a particular node
$ kubectl direct-csi volumes ls --nodes=directcsi-1

# Combine multiple filters using csv
$ kubectl direct-csi vol ls --nodes=directcsi-1,directcsi-2 --status=staged --drives=/dev/nvme0n1
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
	listVolumesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob prefix match for drive status")
}

func listVolumes(ctx context.Context, args []string) error {
	utils.Init()
	dclient := utils.GetDirectCSIClient().DirectCSIDrives()
	vclient := utils.GetDirectCSIClient().DirectCSIVolumes()

	driveList, err := dclient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(driveList.Items) == 0 {
		fmt.Printf("No resource of direct.csi.min.io/v1beta1.%s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	nodeList := nodes
	if len(nodes) == 0 {
		nodeList = []string{"**"}
	}
	if len(nodes) == 1 {
		if nodes[0] == "*" {
			nodeList = []string{"**"}
		}
	}

	filterSet := map[string]struct{}{}
	filterNodes := []directv1beta1.DirectCSIDrive{}
	for _, d := range driveList.Items {
		for _, n := range nodeList {
			if ok, _ := glob.Match(n, d.Status.NodeName); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterNodes = append(filterNodes, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			} else if ok, _ := glob.Match(n+"*", d.Status.NodeName); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterNodes = append(filterNodes, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
		}
	}

	drivesList := drives
	if len(drives) == 0 {
		drivesList = []string{"**"}
	}
	if len(drives) == 1 {
		if drives[0] == "*" {
			drivesList = []string{"**"}
		}
	}

	// reset filterSet
	filterSet = map[string]struct{}{}
	filterDrives := []directv1beta1.DirectCSIDrive{}
	for _, d := range filterNodes {
		for _, n := range drivesList {
			pathTransform := func(in string) string {
				path := strings.ReplaceAll(in, "-part-", "")
				path = strings.ReplaceAll(path, sys.DirectCSIDevRoot+"/", "")
				path = strings.ReplaceAll(path, sys.HostDevRoot+"/", "")
				return filepath.Base(path)
			}

			path := pathTransform(n)
			statusPath := pathTransform(d.Status.Path)

			if ok, _ := glob.Match(path, statusPath); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterDrives = append(filterDrives, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			} else if ok, _ := glob.Match(path+"*", d.Status.RootPartition); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterDrives = append(filterDrives, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
		}
	}

	hasStatus := func(vol *directv1beta1.DirectCSIVolume, status []string) bool {
		statusMatches := 0
		for _, c := range vol.Status.Conditions {
			switch c.Type {
			case string(directv1beta1.DirectCSIVolumeConditionPublished):
				for _, s := range status {
					if strings.ToLower(s) == strings.ToLower(string(directv1beta1.DirectCSIVolumeConditionPublished)) {
						if c.Status == metav1.ConditionTrue {
							statusMatches = statusMatches + 1
						}
					}
				}
			case string(directv1beta1.DirectCSIVolumeConditionStaged):
				for _, s := range status {
					if strings.ToLower(s) == strings.ToLower(string(directv1beta1.DirectCSIVolumeConditionStaged)) {
						if c.Status == metav1.ConditionTrue {
							statusMatches = statusMatches + 1
						}
					}
				}
			}
		}
		return statusMatches == len(status)
	}

	// reset filterSet
	filterSet = map[string]struct{}{}
	vols := []*directv1beta1.DirectCSIVolume{}

	drivePaths := map[string]string{}
	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		return strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
	}

	for _, d := range filterDrives {
		if d.Status.DriveStatus == directv1beta1.DriveStatusUnavailable {
			continue
		}
		drivePaths[d.Name] = driveName(d.Status.Path)
		for _, f := range d.GetFinalizers() {
			if strings.HasPrefix(f, directv1beta1.DirectCSIDriveFinalizerPrefix) {
				name := strings.ReplaceAll(f, directv1beta1.DirectCSIDriveFinalizerPrefix, "")
				vol, err := vclient.Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if hasStatus(vol, status) {
					vols = append(vols, vol)
				}
			}
		}
	}

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"VOLUME",
		"CAPACITY",
		"NODE",
		"DRIVE",
		"",
	})

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
		t.AppendRow([]interface{}{
			v.Name,                                //VOLUME
			emptyOrBytes(v.Status.TotalCapacity),  //CAPACITY
			v.Status.NodeName,                     //SERVER
			driveName(drivePaths[v.Status.Drive]), //DRIVE
		})
	}

	t.Render()
	return nil
}
