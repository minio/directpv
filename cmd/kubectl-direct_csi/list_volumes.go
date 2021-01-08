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
	// "fmt"
	// "os"
	// "sort"
	// "strings"

	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "github.com/dustin/go-humanize"
	// "github.com/fatih/color"
	// "github.com/jedib0t/go-pretty/table"
	// "github.com/jedib0t/go-pretty/text"
	// "github.com/mb0/glob"
	"github.com/spf13/cobra"

	// directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	// "github.com/minio/direct-csi/pkg/sys.
	"github.com/minio/direct-csi/pkg/utils"
)

var listVolumesCmd = &cobra.Command{
	Use:   "list",
	Short: "list volumes in the DirectCSI cluster",
	Long:  "",
	Example: `
# Filter all nvme drives in all nodes 
$ kubectl direct-csi volumes ls --drives=/sys.nvme*

# Filter all new drives 
$ kubectl direct-csi volumes ls --status=new

# Filter all drives from a particular node
$ kubectl direct-csi volumes ls --nodes=directcsi-1

# Combine multiple filters using multi-arg
$ kubectl direct-csi volumes ls --nodes=directcsi-1 --nodes=othernode-2 --status=new

# Combine multiple filters using csv
$ kubectl direct-csi volumes ls --nodes=directcsi-1,othernode-2 --status=new
`,
	RunE: func(c *cobra.Command, args []string) error {
		return listVolumes(c.Context(), args)
	},
	Aliases: []string{
		"ls",
	},
}

func init() {
	listVolumesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glob selector for drive paths")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	listVolumesCmd.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob selector for drive status")
}

func listVolumes(ctx context.Context, args []string) error {
	utils.Init()
	return nil
}
