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
	"regexp"

	"github.com/minio/kubectl-directcsi/util"
	"github.com/minio/minio-go/v6/pkg/set"
	"github.com/minio/minio/pkg/ellipses"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiAddDrivesDesc = `
'add drives' command lets you choose the drives to be managed by DirectCSI.`
	csiAddDrivesExample = `  kubectl directcsi add drives /dev/nvme* --nodes myhost{1...4}`
	defaultFS           = "xfs"
)

func newAddCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add Drives to DirectCSI",
	}
	cmd.AddCommand(newAddDrivesCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	return cmd
}

type csiAddDrivesCmd struct {
	out        io.Writer
	errOut     io.Writer
	output     bool
	force      bool
	nodes      string
	fileSystem string
}

func newAddDrivesCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	c := &csiAddDrivesCmd{out: out, errOut: errOut}

	cmd := &cobra.Command{
		Use:     "drives",
		Short:   "Add Drives to DirectCSI",
		Long:    csiAddDrivesDesc,
		Example: csiAddDrivesExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.run(args)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&c.nodes, "nodes", "n", "", "add drives from these nodes only")
	f.StringVarP(&c.fileSystem, "fs", "f", defaultFS, "filesystem to be formatted. Defaults to 'xfs'")
	f.BoolVarP(&c.force, "force", "", false, "overwrite existing filesystem")

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (c *csiAddDrivesCmd) run(args []string) error {
	ctx := context.Background()

	directCSIClient := util.GetDirectCSIClient()
	drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	if !ellipses.HasEllipses(c.nodes) {
		return fmt.Errorf("please provide --node flag in ellipses format, e.g. `myhost{1...4}`")
	}

	var nodes []string
	if c.nodes != "" {
		pattern, err := ellipses.FindEllipsesPatterns(c.nodes)
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
					drive.DirectCSIOwned = true
					drive.RequestedFormat.Filesystem = c.fileSystem
					drive.RequestedFormat.Force = c.force
					directCSIClient.DirectCSIDrives().Update(ctx, &drive, metav1.UpdateOptions{})
				}
			}
		}
	} else {
		for _, drive := range drives.Items {
			match, _ := regexp.Match(args[0], []byte(drive.Path))
			if match {
				drive.DirectCSIOwned = true
				drive.RequestedFormat.Filesystem = c.fileSystem
				drive.RequestedFormat.Force = c.force
				directCSIClient.DirectCSIDrives().Update(ctx, &drive, metav1.UpdateOptions{})
			}
		}
	}

	return nil
}
