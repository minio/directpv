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
	"sort"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

var listDrivesCmd = &cobra.Command{
	Use:   "list",
	Short: "list drives in the DirectCSI cluster",
	Long:  "",
	Example: `
# Filter all nvme drives in all nodes 
$ kubectl direct-csi drives ls --drives='/dev/nvme*'

# Filter all ready drives 
$ kubectl direct-csi drives ls --status=ready

# Filter all drives from a particular node
$ kubectl direct-csi drives ls --nodes=directcsi-1

# Combine multiple filters using multi-arg
$ kubectl direct-csi drives ls --nodes=directcsi-1 --nodes=othernode-2 --status=available

# Combine multiple filters using csv
$ kubectl direct-csi drives ls --nodes=directcsi-1,othernode-2 --status=ready

# Filter all drives based on access-tier
$ kubectl direct-csi drives drives ls --access-tier="hot"

# Filter all drives with access-tier being set
$ kubectl direct-csi drives drives ls --access-tier="*"
`,
	RunE: func(c *cobra.Command, args []string) error {
		return listDrives(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

var all bool

func init() {
	listDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glob prefix match for drive paths")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob prefix match for node names")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob prefix match for drive status")
	listDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "list all drives (including unavailable)")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "filter based on access-tier")
}

func listDrives(ctx context.Context, args []string) error {
	accessTierSet, err := getAccessTierSet(accessTiers)
	if err != nil {
		return err
	}

	filteredDrives, err := getFilteredDriveList(
		ctx,
		utils.GetDirectCSIClient().DirectCSIDrives(),
		func(drive directcsi.DirectCSIDrive) bool {
			if drive.Status.DriveStatus == directcsi.DriveStatusUnavailable && !all {
				return false
			}
			return drive.MatchGlob(nodes, drives, status) && drive.MatchAccessTier(accessTierSet)
		},
	)
	if err != nil {
		return err
	}

	sort.SliceStable(filteredDrives, func(i, j int) bool {
		d1 := filteredDrives[i]
		d2 := filteredDrives[j]

		if v := strings.Compare(d1.Status.NodeName, d2.Status.NodeName); v != 0 {
			return v < 0
		}

		if v := strings.Compare(d1.Status.Path, d2.Status.Path); v != 0 {
			return v < 0
		}

		return strings.Compare(string(d1.Status.DriveStatus), string(d2.Status.DriveStatus)) < 0
	})

	wrappedDriveList := directcsi.DirectCSIDriveList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "List",
			APIVersion: utils.DirectCSIGroupVersion,
		},
		Items: filteredDrives,
	}
	if yaml || json {
		if err := printer(wrappedDriveList); err != nil {
			klog.ErrorS(err, "error marshaling drives", "format", outputMode)
			return err
		}
		return nil
	}

	headers := func() []interface{} {
		header := []interface{}{
			"DRIVE",
			"CAPACITY",
			"ALLOCATED",
			"FILESYSTEM",
			"VOLUMES",
			"NODE",
			"ACCESS-TIER",
			"STATUS",
			"",
		}
		if wide {
			header = append(header, "DRIVE ID")
		}
		return header
	}()

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row(headers))

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range filteredDrives {
		volumes := len(d.Finalizers)
		if volumes > 0 {
			volumes--
		}

		msg := ""
		for _, c := range d.Status.Conditions {
			if d.Status.DriveStatus == directcsi.DriveStatusReleased {
				// Do not diplay error in case of released drives
				continue
			}
			if c.Type == string(directcsi.DirectCSIDriveConditionInitialized) {
				if c.Status != metav1.ConditionTrue {
					msg = c.Message
					continue
				}
			}
			if c.Type == string(directcsi.DirectCSIDriveConditionOwned) {
				if c.Status != metav1.ConditionTrue {
					msg = c.Message
					continue
				}
			}
		}

		dr := func(val string) string {
			dr := canonicalNameFromPath(val)
			return strings.ReplaceAll("/dev/"+dr, directCSIPartitionInfix, "")
		}(d.Status.Path)
		drStatus := d.Status.DriveStatus
		if msg != "" {
			drStatus = drStatus + "*"
			msg = strings.ReplaceAll(msg, d.Name, "")
			msg = strings.ReplaceAll(msg, sys.GetDirectCSIPath(d.Status.FilesystemUUID), dr)
			msg = strings.ReplaceAll(msg, directCSIPartitionInfix, "")
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
			printableString(d.Status.Filesystem),
			emptyOrVal(volumes), //VOLUMES
			d.Status.NodeName,   //SERVER
			func(drive directcsi.DirectCSIDrive) string {
				if drive.Status.AccessTier == directcsi.AccessTierUnknown {
					return "-"
				}
				return strings.ToLower(string(drive.Status.AccessTier))
			}(d), //ACCESS-TIER
			utils.Bold(drStatus), //STATUS
			msg,                  //MESSAGE
			func() string {
				if wide {
					return d.Name
				}
				return ""
			}(),
		})
	}

	t.Render()
	return nil
}
