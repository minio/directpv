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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

var purgeDrivesCmd = &cobra.Command{
	Use:   "purge",
	Short: utils.BinaryNameTransform("purge detached|lost drives in the {{ . }} cluster"),
	Long:  "",
	Example: utils.BinaryNameTransform(`
# Purge all drives in the cluster
$ kubectl {{ . }} drives purge --all

# Purge 'sdf' drives in all nodes
$ kubectl {{ . }} drives purge --drives '/dev/sdf'

# Purge the lost drives using ellipses notation for drive paths
$ kubectl {{ . }} drives purge --drives '/dev/sd{a...z}'

# Purge the lost drives from selective nodes using ellipses notation for node names
$ kubectl {{ . }} drives purge --nodes 'direct-{1...3}'

# Purge all lost drives from a particular node
$ kubectl {{ . }} drives purge --nodes=direct-1

# Purge all lost drives based on the access-tier set [hot|cold|warm]
$ kubectl {{ . }} drives purge --access-tier=hot

# Combine multiple parameters using multi-arg
$ kubectl {{ . }} drives purge --nodes=direct-1 --nodes=othernode-2

# Combine multiple parameters using csv
$ kubectl {{ . }} drives purge --nodes=direct-1,othernode-2

# Combine multiple parameters using ellipses notations
$ kubectl {{ . }} drives purge --nodes "direct-{3...4}" --drives "/dev/xvd{b...f}"

# Purge a lost drive by it's drive-id
$ kubectl {{ . }} drives purge <drive_id>

# Purge more than one lost drives by their drive-ids
$ kubectl {{ . }} drives purge <drive_id_1> <drive_id_2>
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
		return purgeDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	purgeDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	purgeDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	purgeDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "purge all lost drives")
	purgeDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers,
		"purge based on access-tier set. The possible values are hot|cold|warm")
}

func purgeDrives(ctx context.Context, IDArgs []string) error {
	return processFilteredDrives(
		ctx,
		IDArgs,
		func(drive *directcsi.DirectCSIDrive) bool {
			path := canonicalNameFromPath(drive.Status.Path)
			driveAddr := fmt.Sprintf("%s:/dev/%s", drive.Status.NodeName, path)
			if !utils.IsCondition(drive.Status.Conditions,
				string(directcsi.DirectCSIDriveConditionReady),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIDriveReasonLost),
				string(directcsi.DirectCSIDriveMessageLost)) {
				klog.Errorf("%s is intact. Only lost|detached drive can be purged", utils.Bold(driveAddr))
				return false
			}
			if drive.Status.DriveStatus == directcsi.DriveStatusInUse {
				klog.Errorf("%s is in use. Please purge the corresponding volumes first", utils.Bold(driveAddr))
				return false
			}
			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			drive.Finalizers = []string{}
			utils.UpdateCondition(
				drive.Status.Conditions,
				string(directcsi.DirectCSIDriveConditionOwned),
				metav1.ConditionFalse,
				string(directcsi.DirectCSIDriveReasonAdded),
				"",
			)
			return nil
		},
		func(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
			driveClient := client.GetLatestDirectCSIDriveInterface()
			if _, err := driveClient.Update(ctx, drive, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
				return err
			}
			if err := driveClient.Delete(ctx, drive.Name, metav1.DeleteOptions{}); err != nil {
				if !k8serrors.IsNotFound(err) {
					return err
				}
			}
			return nil
		},
		DrivePurge,
	)
}
