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
	"regexp"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/minio-go/v6/pkg/set"
	"github.com/minio/minio/pkg/ellipses"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiListDrivesDesc = `
list command lists drives status across the storage nodes managed by DirectCSI.`
	csiListDrivesExample = `  kubectl directcsi drives list /dev/nvme* --nodes 'rack*' --all`
)

type csiListDrivesCmd struct {
	output bool
	all    bool
	drives string
	nodes  string
	status string
}

func newDrivesListCmd() *cobra.Command {
	l := &csiListDrivesCmd{}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List drives status across DirectCSI nodes",
		Long:    csiListDrivesDesc,
		Example: csiListDrivesExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return l.run()
		},
	}
	f := cmd.Flags()
	f.StringVarP(&l.drives, "drives", "d", "", "list these drives only")
	f.StringVarP(&l.nodes, "nodes", "n", "", "list drives from particular nodes only")
	f.StringVarP(&l.status, "status", "s", "", "filter by status [in-use, unformatted, new, terminating, unavailable, ready]")
	f.BoolVarP(&l.all, "all", "", false, "list all drives")

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (l *csiListDrivesCmd) run() error {
	utils.Init()

	ctx := context.Background()

	directCSIClient := utils.GetDirectCSIClient()
	drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	volumes, err := directCSIClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all volumes: %v", err)
	}

	if l.nodes != "" && !ellipses.HasEllipses(l.nodes) {
		return fmt.Errorf("please provide --node flag in ellipses format, e.g. `myhost{1...4}`")
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"SERVER", "DRIVES", "STATUS", "VOLUMES", "CAPACITY", "ALLOCATED", "FREE", "FS", "MOUNT", "MODEL"})

	var nodes []string
	if l.nodes != "" {
		pattern, err := ellipses.FindEllipsesPatterns(l.nodes)
		if err != nil {
			return err
		}
		for _, p := range pattern {
			nodes = append(nodes, p.Expand()...)
		}
	}

	nodeSet := set.CreateStringSet(nodes...)
	if !nodeSet.IsEmpty() {
		for _, drive := range drives.Items {
			if nodeSet.Contains(drive.Status.NodeName) {
				match, _ := regexp.Match(l.drives, []byte(drive.Status.Path))
				if match {
					t.AppendRow(table.Row{
						drive.Status.NodeName,
						drive.Status.Path,
						string(drive.Status.DriveStatus),
						strconv.Itoa(len(ListVolumesInDrive(drive, volumes, make([]directv1alpha1.DirectCSIVolume, 0)))),
						humanize.SI(float64(drive.Status.TotalCapacity), "B"),
						humanize.SI(float64(drive.Status.AllocatedCapacity), "B"),
						humanize.SI(float64(drive.Status.FreeCapacity), "B"),
						drive.Status.Filesystem,
						drive.Status.Mountpoint,
						drive.Status.ModelNumber,
					})
				}
			}
		}
	} else {
		for _, drive := range drives.Items {
			match, _ := regexp.Match(l.drives, []byte(drive.Status.Path))
			if match {
				t.AppendRow(table.Row{
					drive.Status.NodeName,
					drive.Status.Path,
					string(drive.Status.DriveStatus),
					strconv.Itoa(len(ListVolumesInDrive(drive, volumes, make([]directv1alpha1.DirectCSIVolume, 0)))),
					humanize.SI(float64(drive.Status.TotalCapacity), "B"),
					humanize.SI(float64(drive.Status.AllocatedCapacity), "B"),
					humanize.SI(float64(drive.Status.FreeCapacity), "B"),
					drive.Status.Filesystem,
					drive.Status.Mountpoint,
					drive.Status.ModelNumber,
				})
			}
		}
	}
	style := table.StyleColoredDark
	style.Color.IndexColumn = text.Colors{text.FgHiBlue, text.BgHiBlack}
	style.Color.Header = text.Colors{text.FgHiBlue, text.BgHiBlack}
	t.SetStyle(style)
	t.Render()

	return nil
}
