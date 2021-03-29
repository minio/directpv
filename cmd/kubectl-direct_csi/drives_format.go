/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
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
	"sort"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/golang/glog"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const XFS = "xfs"

var (
	force = false
)

var formatDrivesCmd = &cobra.Command{
	Use:   "format",
	Short: "format drives in the DirectCSI cluster",
	Long:  "",
	Example: `
# Format all available drives in the cluster
$ kubectl direct-csi drives format --all

# Format all nvme drives in all nodes 
$ kubectl direct-csi drives format --drives=/dev/nvme*

# Format all drives from a particular node
$ kubectl direct-csi drives format --nodes=directcsi-1

# Format all drives based on the access-tier set [hot|cold|warm]
$ kubectl direct-csi drives format --access-tier=hot

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives format --nodes=directcsi-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl direct-csi drives format --nodes=directcsi-1,othernode-2 --status=available
`,
	RunE: func(c *cobra.Command, args []string) error {
		return formatDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glog selector for drive paths")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	formatDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "format all available drives")

	formatDrivesCmd.PersistentFlags().BoolVarP(&force, "force", "f", force, "force format a drive even if a FS is already present")

	formatDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "format based on access-tier set. The possible values are [hot,cold,warm] ")
}

func formatDrives(ctx context.Context, args []string) error {
	dryRun := viper.GetBool(dryRunFlagName)

	if !all {
		if len(drives) == 0 && len(nodes) == 0 && len(accessTiers) == 0 {
			return fmt.Errorf("atleast one of '%s', '%s' or '%s' should be specified", utils.Bold("--all"), utils.Bold("--drives"), utils.Bold("--nodes"))
		}
	}
	utils.Init()

	directClient := utils.GetDirectCSIClient()
	driveList, err := directClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(driveList.Items) == 0 {
		glog.Errorf("No resource of %s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	volList, err := directClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
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

	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		dr = strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
		return strings.ReplaceAll(dr, "-part-", "")
	}

	updatedFilterDrives := []*directcsi.DirectCSIDrive{}
	for _, d := range filterDrives {
		if d.Status.DriveStatus == directcsi.DriveStatusUnavailable {
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusInUse {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			glog.Errorf("%s in use. Cannot be formatted", utils.Bold(driveAddr))
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusReady {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			glog.Errorf("%s already owned and managed. Use %s to overwrite", utils.Bold(driveAddr), utils.Bold("--force"))
			continue
		}
		if d.Status.Filesystem != "" && !force {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			glog.Errorf("%s already has a fs. Use %s to overwrite", utils.Bold(driveAddr), utils.Bold("--force"))
			continue
		}

		d.Spec.DirectCSIOwned = true
		d.Spec.RequestedFormat = &directcsi.RequestedFormat{
			Filesystem: XFS,
			Force:      force,
		}
		if dryRun {
			if err := utils.LogYAML(d); err != nil {
				return err
			}
			continue
		}
		updated, err := directClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		updatedFilterDrives = append(updatedFilterDrives, updated)
	}

	if dryRun {
		return nil
	}

	sort.SliceStable(updatedFilterDrives, func(i, j int) bool {
		d1 := updatedFilterDrives[i]
		d2 := updatedFilterDrives[j]

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
		"",
	})

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range updatedFilterDrives {
		volumes := 0
		for _, v := range volList.Items {
			if v.Status.Drive == d.Name {
				volumes++
			}
		}

		msg := ""
		dr := func(val string) string {
			dr := driveName(val)
			col := red(dot)
			for _, c := range d.Status.Conditions {
				if c.Type == string(directcsi.DirectCSIDriveConditionOwned) {
					if c.Status == metav1.ConditionTrue {
						col = green(dot)
					}
					c.Message = msg
				}
			}
			return strings.ReplaceAll(col+" "+dr, "-part-", "")
		}(d.Status.Path)
		drStatus := d.Status.DriveStatus
		if msg != "" {
			drStatus = drStatus + "*"
			msg = strings.ReplaceAll(msg, d.Name, "")
			msg = strings.ReplaceAll(msg, "/var/lib/direct-csi/devices", "/dev")
			msg = strings.Split(msg, "\n")[0]
		}
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
			utils.Bold(drStatus),                     //STATUS
			msg,                                      //MESSSAGE
		})
	}

	//t.Render()
	return nil
}
