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
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/mb0/glob"
	"github.com/spf13/cobra"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/dev"
	"github.com/minio/direct-csi/pkg/utils"
)

var (
	drives = []string{}
	nodes  = []string{}
	status = []string{}
)

var listDrivesCmd = &cobra.Command{
	Use:   "list",
	Short: "list drives in the DirectCSI cluster",
	Long:  "",
	Example: `
# Filter all nvme drives in all nodes 
$ kubectl direct-csi drives list --drives=/dev/nvme*

# Filter all new drives 
$ kubectl direct-csi drives list --status=new

# Filter all drives from a particular node
$ kubectl direct-csi drives list --nodes=directcsi-1

# Combine multiple filters using multi-arg
$ kubectl direct-csi drives list --nodes=directcsi-1 --nodes=othernode-2 --status=new

# Combine multiple filters using csv
$ kubectl direct-csi drives list --nodes=directcsi-1,othernode-2 --status=new
`,
	RunE: func(c *cobra.Command, args []string) error {
		return listDrives(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glob selector for drive paths")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob selector for drive status")
}

func listDrives(ctx context.Context, args []string) error {
	utils.Init()

	bold := color.New(color.Bold).SprintFunc()
	directClient := utils.GetDirectCSIClient()

	driveList, err := directClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(driveList.Items) == 0 {
		fmt.Printf("No resource of direct.csi.min.io/v1alpha1.%s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	volList, err := directClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
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
	filterNodes := []directv1alpha1.DirectCSIDrive{}
	for _, d := range driveList.Items {
		for _, n := range nodeList {
			if ok, _ := glob.Match(n, d.Status.NodeName); ok {
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
	filterDrives := []directv1alpha1.DirectCSIDrive{}
	for _, d := range filterNodes {
		for _, n := range drivesList {
			if ok, _ := glob.Match(n, d.Status.Path); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterDrives = append(filterDrives, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
			if ok, _ := glob.Match(n, strings.ReplaceAll(d.Status.Path, dev.DevRoot, "/dev")); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterDrives = append(filterDrives, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
		}
	}

	statusesList := status
	if len(status) == 0 {
		statusesList = []string{"**"}
	}
	if len(status) == 1 {
		if status[0] == "*" {
			statusesList = []string{"**"}
		}
	}

	// reset filterSet
	filterSet = map[string]struct{}{}
	filterStatus := []directv1alpha1.DirectCSIDrive{}
	for _, d := range filterDrives {
		for _, n := range statusesList {
			if ok, _ := glob.Match(n, string(d.Status.DriveStatus)); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterStatus = append(filterStatus, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
		}
	}

	totalOwnedDrives := 0
	totalDrives := len(filterStatus)
	for _, d := range filterStatus {
		if d.Spec.DirectCSIOwned {
			totalOwnedDrives++
		}
	}

	sort.SliceStable(filterStatus, func(i, j int) bool {
		d1 := filterStatus[i]
		d2 := filterStatus[j]

		if v := strings.Compare(d1.Status.NodeName, d2.Status.NodeName); v != 0 {
			return v < 0
		}

		if v := strings.Compare(d1.Status.Path, d2.Status.Path); v != 0 {
			return v < 0
		}

		return strings.Compare(string(d1.Status.DriveStatus), string(d2.Status.DriveStatus)) < 0
	})

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"SERVER", "DRIVE", "STATUS", "VOLUMES", "CAPACITY", "ALLOCATED", "FREE", "FS", "MOUNT"})

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range filterStatus {
		volumes := 0
		for _, v := range volList.Items {
			if v.Status.OwnerDrive == d.Name {
				volumes++
			}
		}

		t.AppendRow([]interface{}{
			d.Status.NodeName, //SERVER
			strings.ReplaceAll(d.Status.Path, dev.DevRoot, "/dev"), //DRIVE
			string(d.Status.DriveStatus),                           //STATUS
			volumes,                                                //VOLUMES
			humanize.Bytes(uint64(d.Status.TotalCapacity)),         //CAPACITY
			humanize.Bytes(uint64(d.Status.AllocatedCapacity)),     //ALLOCATED
			humanize.Bytes(uint64(d.Status.FreeCapacity)),          //FREE
			func(fs string) string {
				if fs == "" {
					return "-"
				}
				return fs
			}(d.Status.Filesystem), //FS
			func(mountpoint string) string {
				if mountpoint == "" {
					return "-"
				}
				newMP := mountpoint
				//newMP := strings.ReplaceAll(d.Status.Mountpoint, "/var/lib/direct-csi/mnt/", "${DIRECT_CSI_ROOT}/mnt/")
				if len(newMP) > 48 {
					newMP = newMP[:48]
				}
				return newMP + "..."
			}(d.Status.Mountpoint), //MOUNT
		})
	}

	t.Render()
	fmt.Println()
	fmt.Printf("(%s/%s) Drives managed by direct-csi\n", bold(fmt.Sprintf("%d", totalOwnedDrives)), bold(fmt.Sprintf("%d", totalDrives)))

	return nil
}
