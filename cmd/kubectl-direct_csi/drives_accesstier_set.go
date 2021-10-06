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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/spf13/cobra"
)

var accessTierSet = &cobra.Command{
	Use:   "set [hot|cold|warm]",
	Short: "tag DirectCSI drive(s) based on their access-tiers [hot,cold,warm]",
	Long:  "",
	Example: `
# Sets the 'access-tier:cold' tag to all the 'Available' DirectCSI drives 
$ kubectl direct-csi drives access-tier set cold --all

# Sets the 'access-tier:warm' tag to 'sdf' drives in all nodes
$ kubectl direct-csi drives access-tier set warm --drives '/dev/sdf'

# Sets the 'access-tier:hot' tag to selective drives using ellipses expander
$ kubectl direct-csi drives access-tier set hot --drives '/dev/sd{a...z}'

# Sets the 'access-tier:hot' tag to drives from selective nodes using ellipses expander
$ kubectl direct-csi drives access-tier set hot --nodes 'directcsi-{1...3}'

# Sets the 'access-tier:hot' tag to all drives from a particular node
$ kubectl direct-csi drives access-tier set hot --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives access-tier set hot --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl direct-csi drives access-tier set hot --nodes=directcsi-1,othernode-2 --status=ready
`,
	RunE: func(c *cobra.Command, args []string) error {
		return setAccessTier(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	accessTierSet.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "ellipses expander for drive paths")
	accessTierSet.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "ellipses expander for node names")
	accessTierSet.PersistentFlags().BoolVarP(&all, "all", "a", all, "tag all available drives")
	accessTierSet.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "match based on drive status ['inuse','available','unavailable','ready','terminating','released']")
}

func setAccessTier(ctx context.Context, accessTierArg []string) error {
	if !all {
		if len(drives) == 0 && len(nodes) == 0 && len(status) == 0 {
			return fmt.Errorf("atleast one of '%s', '%s', '%s' or '%s' should be specified",
				utils.Bold("--all"),
				utils.Bold("--drives"),
				utils.Bold("--nodes"),
				utils.Bold("--status"))
		}
	}

	if len(accessTierArg) != 1 {
		return fmt.Errorf("Invalid input arguments. Please use '%s' for examples to set access-tiers", utils.Bold("--help"))
	}

	accessTier, err := utils.ValidateAccessTier(accessTierArg[0])
	if err != nil {
		return err
	}

	driveStatusList, err := getDriveStatusList(status)
	if err != nil {
		return err
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	return processFilteredDrives(
		ctx,
		utils.GetDirectCSIClient().DirectCSIDrives(),
		nil,
		func(drive *directcsi.DirectCSIDrive) bool {
			if len(driveStatusList) > 0 {
				return drive.MatchDriveStatus(driveStatusList)
			}
			return drive.Status.DriveStatus != directcsi.DriveStatusUnavailable
		},
		func(drive *directcsi.DirectCSIDrive) error {
			drive.Status.AccessTier = accessTier
			utils.SetAccessTierLabel(drive, accessTier)
			return nil
		},
		defaultDriveUpdateFunc,
	)
}
