/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
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
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var accessTierUnset = &cobra.Command{
	Use:   "unset",
	Short: "remove the access-tier tag from the DirectCSI drive(s)",
	Long:  "",
	Example: `
# Unsets the 'access-tier' tag on all the 'Available' DirectCSI drives 
$ kubectl direct-csi drives access-tier unset --all

# Unsets the 'access-tier' based on the tier value set
$ kubectl direct-csi drives access-tier unset --access-tier=warm

# Unsets the 'access-tier' on selective drives using ellipses notation for drive paths
$ kubectl direct-csi drives access-tier unset --drives '/dev/sd{a...z}'

# Unsets the 'access-tier' on drives from selective nodes using ellipses notation for node names
$ kubectl direct-csi drives access-tier unset --nodes 'directcsi-{1...3}'

# Unsets the 'access-tier' tag on all the drives from a particular node
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1,othernode-2 --access-tier=hot
`,
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

	directCSIClient := client.GetDirectCSIClient()
	return processFilteredDrives(
		ctx,
		directCSIClient.DirectCSIDrives(),
		nil,
		nil,
		func(drive *directcsi.DirectCSIDrive) error {
			setDriveAccessTier(drive, directcsi.AccessTierUnknown)
			return nil
		},
		defaultDriveUpdateFunc(directCSIClient),
		UnSetAcessTier,
	)
}
