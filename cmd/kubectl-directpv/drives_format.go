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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"

	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

const xfs = "xfs"

var (
	force = false
)

var formatDrivesCmd = &cobra.Command{
	Use:   "format",
	Short: utils.BinaryNameTransform("format drives in the {{ . }} cluster"),
	Long:  "",
	Example: utils.BinaryNameTransform(`
# Format all available drives in the cluster
$ kubectl {{ . }} drives format --all

# Format the 'sdf' drives in all nodes
$ kubectl {{ . }} drives format --drives '/dev/sdf'

# Format the selective drives using ellipses notation for drive paths
$ kubectl {{ . }} drives format --drives '/dev/sd{a...z}'

# Format the drives from selective nodes using ellipses notation for node names
$ kubectl {{ . }} drives format --nodes 'direct-{1...3}'

# Format all drives from a particular node
$ kubectl {{ . }} drives format --nodes=direct-1

# Format all drives based on the access-tier set [hot|cold|warm]
$ kubectl {{ . }} drives format --access-tier=hot

# Combine multiple parameters using multi-arg
$ kubectl {{ . }} drives format --nodes=direct-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl {{ . }} drives format --nodes=direct-1,othernode-2 --status=available

# Combine multiple parameters using ellipses notations
$ kubectl {{ . }} drives format --nodes "direct-{3...4}" --drives "/dev/xvd{b...f}"

# Format a drive by it's drive-id
$ kubectl {{ . }} drives format <drive_id>

# Format more than one drive by their drive-ids
$ kubectl {{ . }} drives format <drive_id_1> <drive_id_2>
`),
	RunE: func(c *cobra.Command, args []string) error {
		if !all {
			if len(drives) == 0 && len(nodes) == 0 && len(accessTiers) == 0 && len(args) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s' or '%s' must be specified",
					utils.Bold("--all"),
					utils.Bold("--drives"),
					utils.Bold("--nodes"))
			}
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
		return formatDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	formatDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "format all available drives")
	formatDrivesCmd.PersistentFlags().BoolVarP(&force, "force", "f", force, "force format a drive even if a FS is already present")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers,
		"format based on access-tier set. The possible values are hot|cold|warm")
}

func formatDrives(ctx context.Context, IDArgs []string) error {
	return processFilteredDrives(
		ctx,
		IDArgs,
		func(drive *directcsi.DirectCSIDrive) bool {
			if drive.Status.DriveStatus == directcsi.DriveStatusUnavailable {
				return false
			}

			path := canonicalNameFromPath(drive.Status.Path)
			driveAddr := fmt.Sprintf("%s:/dev/%s", drive.Status.NodeName, path)

			if drive.Status.DriveStatus == directcsi.DriveStatusInUse {
				klog.Errorf("%s is in use. Cannot be formatted", utils.Bold(driveAddr))
				return false
			}

			if drive.Status.DriveStatus == directcsi.DriveStatusReady {
				klog.Errorf("%s is already owned and managed", utils.Bold(driveAddr))
				return false
			}

			if drive.Status.Filesystem != "" && !force {
				klog.Errorf("%s already has a fs. Use %s to overwrite", utils.Bold(driveAddr), utils.Bold("--force"))
				return false
			}

			if drive.Status.DriveStatus == directcsi.DriveStatusReleased {
				klog.Errorf(utils.BinaryNameTransform("%s is in 'released' state. Use 'kubectl {{ . }} drives unrelease --drive %s --nodes %s' before formatting"),
					utils.Bold(driveAddr), path, drive.Status.NodeName)
				return false
			}

			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			drive.Spec.DirectCSIOwned = true
			drive.Spec.RequestedFormat = &directcsi.RequestedFormat{
				Filesystem: xfs,
				Force:      force,
			}
			return nil
		},
		defaultDriveUpdateFunc(),
		Format,
	)
}
