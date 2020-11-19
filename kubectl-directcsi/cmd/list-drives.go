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

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/minio/kubectl-directcsi/util"
	"github.com/minio/minio-go/v6/pkg/set"
	"github.com/minio/minio/pkg/ellipses"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiListDrivesDesc = `
 list command lists drives status across the storage nodes managed by DirectCSI.`
	csiListDrivesExample = `  kubectl directcsi drives list /dev/nvme* --nodes 'rack*' --all`
)

type csiListDrivesCmd struct {
	out    io.Writer
	errOut io.Writer
	output bool
	all    bool
	nodes  string
	status string
}

func newDrivesListCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	l := &csiListDrivesCmd{out: out, errOut: errOut}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List drives status across DirectCSI nodes",
		Long:    csiListDrivesDesc,
		Example: csiListDrivesExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return l.run(args)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&l.nodes, "nodes", "n", "", "add drives from these nodes only")
	f.StringVarP(&l.status, "status", "s", "", "filter by status [new, ignore, online, offline]")
	f.BoolVarP(&l.all, "all", "", false, "list all drives")

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (l *csiListDrivesCmd) run(args []string) error {
	ctx := context.Background()

	directCSIClient := util.GetDirectCSIClient()
	drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	volumes, err := directCSIClient.DirectCSIVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	if !ellipses.HasEllipses(l.nodes) {
		return fmt.Errorf("please provide --node flag in ellipses format, e.g. `myhost{1...4}`")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"DRIVES", "STATUS", "VOLUMES", "ALLOCATED", "CAPACITY", "FREE", "FS", "MOUNT", "MODEL"})

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
			if nodeSet.Contains(drive.OwnerNode) {
				match, _ := regexp.Match(args[0], []byte(drive.Path))
				if match {
					table.Append([]string{
						drive.OwnerNode + ":" + drive.Path,
						string(drive.DriveStatus),
						strconv.Itoa(len(util.ListVolumesInDrive(drive, volumes))),
						strconv.FormatInt(drive.AllocatedCapacity, 10),
						strconv.FormatInt(drive.TotalCapacity, 10),
						strconv.FormatInt(drive.FreeCapacity, 10),
						drive.Filesystem,
						drive.Mountpoint,
						drive.ModelNumber,
					})
				}
			}
		}
	} else {
		for _, drive := range drives.Items {
			match, _ := regexp.Match(args[0], []byte(drive.Path))
			if match {
				table.Append([]string{
					drive.OwnerNode + ":" + drive.Path,
					string(drive.DriveStatus),
					strconv.Itoa(len(util.ListVolumesInDrive(drive, volumes))),
					strconv.FormatInt(drive.AllocatedCapacity, 10),
					strconv.FormatInt(drive.TotalCapacity, 10),
					strconv.FormatInt(drive.FreeCapacity, 10),
					drive.Filesystem,
					drive.Mountpoint,
					drive.ModelNumber,
				})
			}
		}
	}
	table.Render()

	return nil
}
