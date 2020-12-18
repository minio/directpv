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
	"regexp"
	"strings"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	v1alpha1 "github.com/minio/direct-csi/pkg/clientset/typed/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/minio-go/v6/pkg/set"
	"github.com/minio/minio/pkg/ellipses"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiAddDrivesDesc = `
add command lets you add drives to be managed by DirectCSI.`
	csiAddDrivesExample = `  kubectl directcsi drives add /dev/nvme* --nodes myhost{1...4}`
	defaultFS           = "xfs"
)

type csiAddDrivesCmd struct {
	output       bool
	force        bool
	nodes        string
	fileSystem   string
	mountOptions string
}

func newDrivesAddCmd() *cobra.Command {
	c := &csiAddDrivesCmd{}

	cmd := &cobra.Command{
		Use:     "add",
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
	f.StringVarP(&c.fileSystem, "fs", "f", defaultFS, "filesystem to be formatted")
	f.StringVarP(&c.mountOptions, "mountOptions", "m", "", "mount options in csv format, e.g. 'noatime,nodiratime'")
	f.BoolVarP(&c.force, "force", "", false, "overwrite existing filesystem")

	return cmd
}

// run initializes local config and installs MinIO Operator to Kubernetes cluster.
func (c *csiAddDrivesCmd) run(args []string) error {
	ctx := context.Background()

	utils.Init()
	directCSIClient := utils.GetDirectCSIClient()
	drives, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list all drives: %v", err)
	}

	if !ellipses.HasEllipses(c.nodes) {
		return fmt.Errorf("please provide --nodes flag in ellipses format, e.g. `myhost{1...4}`")
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
			if nodeSet.Contains(drive.Status.NodeName) {
				match, _ := regexp.Match(args[0], []byte(drive.Status.Path))
				if match {
					c.updateDrive(ctx, drive, directCSIClient)
				}
			}
		}
	} else {
		for _, drive := range drives.Items {
			match, _ := regexp.Match(args[0], []byte(drive.Status.Path))
			if match {
				c.updateDrive(ctx, drive, directCSIClient)
			}
		}
	}

	return nil
}

func (c *csiAddDrivesCmd) updateDrive(ctx context.Context, d directv1alpha1.DirectCSIDrive, client v1alpha1.DirectV1alpha1Interface) {
	d.Spec.DirectCSIOwned = true
	d.Spec.RequestedFormat.Filesystem = c.fileSystem
	d.Spec.RequestedFormat.Force = c.force
	d.Spec.RequestedFormat.Mountoptions = strings.Split(c.mountOptions, ",")
	client.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{})
}
