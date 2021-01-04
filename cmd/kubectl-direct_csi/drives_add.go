/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/mb0/glob"
	"github.com/spf13/cobra"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"
)

const XFS = "xfs"

var (
	force = false
)

var addDrivesCmd = &cobra.Command{
	Use:   "add",
	Short: "add drives to the DirectCSI cluster",
	Long:  "",
	Example: `
# Add all available drives in the cluster
$ kubectl direct-csi drives add

# Add all nvme drives in all nodes 
$ kubectl direct-csi drives add --drives=/sys.nvme*

# Add all new drives
$ kubectl direct-csi drives add --status=new

# Add all drives from a particular node
$ kubectl direct-csi drives add --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives add --nodes=directcsi-1 --nodes=othernode-2 --status=new

# Combine multiple parameters using csv
$ kubectl direct-csi drives add --nodes=directcsi-1,othernode-2 --status=new
`,
	RunE: func(c *cobra.Command, args []string) error {
		return addDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	addDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glog selector for drive paths")
	addDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	addDrivesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob selector for drive status")
	addDrivesCmd.PersistentFlags().BoolVarP(&force, "force", "f", force, "force format a drive even if a FS is already present")
}

func addDrives(ctx context.Context, args []string) error {
	utils.Init()

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
	filterDrives := []directv1alpha1.DirectCSIDrive{}
	for _, d := range filterNodes {
		for _, n := range drivesList {
			pathTransform := func(in string) string {
				path := strings.ReplaceAll(in, "-part-", "")
				path = strings.ReplaceAll(path, sys.DirectCSIDevRoot, "")
				path = strings.ReplaceAll(path, sys.HostDevRoot, "")
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
			} else if ok, _ := glob.Match(n+"*", string(d.Status.DriveStatus)); ok {
				if _, ok := filterSet[d.Name]; !ok {
					filterStatus = append(filterStatus, d)
					filterSet[d.Name] = struct{}{}
				}
				continue
			}
		}
	}

	updatedFilterStatus := []*directv1alpha1.DirectCSIDrive{}
	for _, d := range filterStatus {
		if d.Status.DriveStatus == directv1alpha1.DriveStatusUnavailable {
			continue
		}
		d.Spec.DirectCSIOwned = true
		d.Spec.RequestedFormat = &directv1alpha1.RequestedFormat{
			Filesystem: XFS,
			Force:      force,
		}
		updated, err := directClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		updatedFilterStatus = append(updatedFilterStatus, updated)
	}

	sort.SliceStable(updatedFilterStatus, func(i, j int) bool {
		d1 := updatedFilterStatus[i]
		d2 := updatedFilterStatus[j]

		if v := strings.Compare(d1.Status.NodeName, d2.Status.NodeName); v != 0 {
			return v < 0
		}

		if v := strings.Compare(d1.Status.Path, d2.Status.Path); v != 0 {
			return v < 0
		}

		return strings.Compare(string(d1.Status.DriveStatus), string(d2.Status.DriveStatus)) < 0
	})

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"DRIVE",
		"CAPACITY",
		"ALLOCATED",
		"VOLUMES",
		"NODE",
		"STATUS",
	})

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range updatedFilterStatus {
		volumes := 0
		for _, v := range volList.Items {
			if v.Status.OwnerDrive == d.Name {
				volumes++
			}
		}

		dr := func(val string) string {
			dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot, "")
			dr = strings.ReplaceAll(dr, sys.HostDevRoot, "")
			col := red(dot)
			for _, c := range d.Status.Conditions {
				if c.Type == string(directv1alpha1.DirectCSIDriveConditionOwned) {
					if c.Status == metav1.ConditionTrue {
						col = green(dot)
					}
				}
			}
			return strings.ReplaceAll(col+" "+dr, "-part-", "")
		}(d.Status.Path)
		drStatus := d.Status.DriveStatus
		emptyOrVal := func(val int) string {
			if val == 0 {
				return "-"
			}
			return fmt.Sprintf("%d", val)
		}
		emptyOrBytes := func(val int64) string {
			if val == 0 {
				return "-"
			}
			return humanize.IBytes(uint64(val))
		}
		t.AppendRow([]interface{}{
			dr,                                       //DRIVE
			emptyOrBytes(d.Status.TotalCapacity),     //CAPACITY
			emptyOrBytes(d.Status.AllocatedCapacity), //ALLOCATED
			emptyOrVal(volumes),                      //VOLUMES
			d.Status.NodeName,                        //SERVER
			drStatus,                                 //STATUS
		})
	}

	t.Render()
	return nil
}
