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
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var accessTierUnset = &cobra.Command{
	Use:   "unset",
	Short: utils.BinaryNameTransform("remove the access-tier tag from the {{ . }} drive(s)"),
	Long:  "",
	Example: utils.BinaryNameTransform(`
# Unsets the 'access-tier' tag on all the 'Available' {{ . }} drives 
$ kubectl {{ . }} drives access-tier unset --all

# Unsets the 'access-tier' based on the tier value set
$ kubectl {{ . }} drives access-tier unset --access-tier=warm

# Unsets the 'access-tier' on selective drives using ellipses notation for drive paths
$ kubectl {{ . }} drives access-tier unset --drives '/dev/sd{a...z}'

# Unsets the 'access-tier' on drives from selective nodes using ellipses notation for node names
$ kubectl {{ . }} drives access-tier unset --nodes 'directcsi-{1...3}'

# Unsets the 'access-tier' tag on all the drives from a particular node
$ kubectl {{ . }} drives access-tier unset --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl {{ . }} drives access-tier unset --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl {{ . }} drives access-tier unset --nodes=directcsi-1,othernode-2 --access-tier=hot
`),
	RunE: func(c *cobra.Command, _ []string) error {
		if !all {
			if len(drives) == 0 && len(nodes) == 0 && len(status) == 0 && len(accessTiers) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s', '%s', '%s', or '%s' must be specified",
					utils.Bold("--all"),
					utils.Bold("--drives"),
					utils.Bold("--nodes"),
					utils.Bold("--status"),
					utils.Bold("--access-tier"))
			}
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 || len(statusGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}

		return unsetAccessTier(c.Context())
	},
	Aliases: []string{},
}

func init() {
	accessTierUnset.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	accessTierUnset.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	accessTierUnset.PersistentFlags().BoolVarP(&all, "all", "a", all, "untag all available drives")
	accessTierUnset.PersistentFlags().StringSliceVarP(&status, "status", "s", status, fmt.Sprintf("match based on drive status [%s]", strings.Join(directcsi.SupportedStatusSelectorValues(), ", ")))
	accessTierUnset.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "match based on access-tier set. The possible values are [hot,cold,warm] ")
}

func unsetAccessTier(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	return processFilteredDrives(
		ctx,
		nil,
		nil,
		func(drive *directcsi.DirectCSIDrive) error {
			setDriveAccessTier(drive, directcsi.AccessTierUnknown)
			return nil
		},
		defaultDriveUpdateFunc(),
		UnSetAcessTier,
	)
}
