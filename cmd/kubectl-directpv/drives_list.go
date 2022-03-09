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
	"fmt"
	"os"
	"sort"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

var listDrivesCmd = &cobra.Command{
	Use:   "list",
	Short: utils.BinaryNameTransform("list drives in the {{ . }} cluster"),
	Long:  "",
	Example: utils.BinaryNameTransform(`
# List all drives
$ kubectl {{ . }} drives ls

# List all drives (including 'unavailable' drives)
$ kubectl {{ . }} drives ls --all

# Filter all ready drives 
$ kubectl {{ . }} drives ls --status=ready

# Filter all drives from a particular node
$ kubectl {{ . }} drives ls --nodes=direct-1

# Combine multiple filters using multi-arg
$ kubectl {{ . }} drives ls --nodes=direct-1 --nodes=othernode-2 --status=available

# Combine multiple filters using csv
$ kubectl {{ . }} drives ls --nodes=direct-1,othernode-2 --status=ready

# Filter all drives based on access-tier
$ kubectl {{ . }} drives drives ls --access-tier="hot"

# Filter all drives with access-tier being set
$ kubectl {{ . }} drives drives ls --access-tier="*"

# Filter drives by ellipses notation for drive paths and nodes
$ kubectl {{ . }} drives ls --drives='/dev/xvd{a...d}' --nodes='node-{1...4}'
`),
	RunE: func(c *cobra.Command, args []string) error {
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 || len(statusGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
		return listDrives(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

var all bool

func init() {
	listDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, fmt.Sprintf("match based on drive status [%s]", strings.Join(directcsi.SupportedStatusSelectorValues(), ", ")))
	listDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "list all drives (including unavailable)")
	listDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "match based on access-tier")
}

func getModel(drive directcsi.DirectCSIDrive) string {
	var result string

	switch {
	case drive.Status.DMUUID != "":
		result = drive.Status.DMName
	case drive.Status.MDUUID != "":
		result = "Linux RAID"
	default:
		vendor := strings.TrimSpace(drive.Status.Vendor)
		model := strings.TrimSpace(drive.Status.ModelNumber)
		switch {
		case vendor == "" || strings.Contains(model, vendor):
			result = model
		case model == "" || strings.Contains(vendor, model):
			result = vendor
		default:
			result = vendor + " " + model
		}
	}

	if drive.Status.PartitionNum <= 0 {
		return result
	}

	if result == "" {
		return "PART"
	}

	return result + " PART"
}

func listDrives(ctx context.Context, args []string) error {
	filteredDrives, err := getFilteredDriveList(
		ctx,
		func(drive directcsi.DirectCSIDrive) bool {
			if len(driveStatusList) > 0 {
				return drive.MatchDriveStatus(driveStatusList)
			}
			return all || len(statusGlobs) > 0 || drive.Status.DriveStatus != directcsi.DriveStatusUnavailable
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
			APIVersion: string(utils.DirectCSIVersionLabelKey),
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

	headers := []interface{}{
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
		headers = append(headers, "DRIVE ID", "MODEL")
	}

	text.DisableColors()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if !noHeaders {
		t.AppendHeader(table.Row(headers))
	}

	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)

	for _, d := range filteredDrives {
		drive := strings.ReplaceAll(
			"/dev/"+canonicalNameFromPath(d.Status.Path),
			directCSIPartitionInfix,
			"",
		)

		volumes := "-"
		if len(d.Finalizers) > 1 {
			volumes = fmt.Sprintf("%v", len(d.Finalizers)-1)
		}

		accessTier := "-"
		if d.Status.AccessTier != directcsi.AccessTierUnknown {
			accessTier = strings.ToLower(string(d.Status.AccessTier))
		}

		status := d.Status.DriveStatus
		msg := ""
		for _, c := range d.Status.Conditions {
			switch c.Type {
			case string(directcsi.DirectCSIDriveConditionInitialized), string(directcsi.DirectCSIDriveConditionOwned), string(directcsi.DirectCSIDriveConditionReady):
				if c.Status != metav1.ConditionTrue {
					msg = c.Message
					if msg != "" {
						status = d.Status.DriveStatus + "*"
						msg = strings.ReplaceAll(msg, d.Name, "")
						msg = strings.ReplaceAll(msg, getDirectCSIPath(d.Status.FilesystemUUID), drive)
						msg = strings.ReplaceAll(msg, directCSIPartitionInfix, "")
						msg = strings.Split(msg, "\n")[0]
					}
				}
			}
		}
		var allocatedCapacity int64
		if status == directcsi.DriveStatusInUse {
			allocatedCapacity = d.Status.AllocatedCapacity
		}

		output := []interface{}{
			drive,
			printableBytes(d.Status.TotalCapacity),
			printableBytes(allocatedCapacity),
			printableString(d.Status.Filesystem),
			volumes,
			d.Status.NodeName,
			accessTier,
			utils.Bold(status),
			msg,
		}

		if wide {
			output = append(output, d.Name, printableString(getModel(d)))
		}

		t.AppendRow(output)
	}

	t.Render()
	return nil
}
