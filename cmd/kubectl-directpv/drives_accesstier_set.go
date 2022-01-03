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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/utils"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var accessTierSet = &cobra.Command{
	Use:   "set [hot|cold|warm]",
	Short: "tag DirectPV drive(s) based on their access-tiers [hot,cold,warm]",
	Long:  "",
	Example: `
# Sets the 'access-tier:cold' tag to all the 'Available' DirectPV drives 
$ kubectl directpv drives access-tier set cold --all

# Sets the 'access-tier:warm' tag to 'sdf' drives in all nodes
$ kubectl directpv drives access-tier set warm --drives '/dev/sdf'

# Sets the 'access-tier:hot' tag to selective drives using ellipses notation for drive paths
$ kubectl directpv drives access-tier set hot --drives '/dev/sd{a...z}'

# Sets the 'access-tier:hot' tag to drives from selective nodes using ellipses notation for node names
$ kubectl directpv drives access-tier set hot --nodes 'directpv-{1...3}'

# Sets the 'access-tier:hot' tag to all drives from a particular node
$ kubectl directpv drives access-tier set hot --nodes=directpv-1

# Combine multiple parameters using multi-arg
$ kubectl directpv drives access-tier set hot --nodes=directpv-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl directpv drives access-tier set hot --nodes=directpv-1,othernode-2 --status=ready
`,
	RunE: func(c *cobra.Command, args []string) error {
		if !all {
			if len(drives) == 0 && len(nodes) == 0 && len(status) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s', '%s' or '%s' must be specified",
					utils.Bold("--all"),
					utils.Bold("--drives"),
					utils.Bold("--nodes"),
					utils.Bold("--status"))
			}
		}
		if len(args) != 1 {
			return fmt.Errorf("only one access tier must be specified. please use '%s' for examples to set access-tier", utils.Bold("--help"))
		}
		accessTier, err := directcsi.ToAccessTier(args[0])
		if err != nil {
			return err
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 || len(statusGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
		return setAccessTier(c.Context(), accessTier)
	},
	Aliases: []string{},
}

func init() {
	accessTierSet.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	accessTierSet.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	accessTierSet.PersistentFlags().BoolVarP(&all, "all", "a", all, "tag all available drives")
	accessTierSet.PersistentFlags().StringSliceVarP(&status, "status", "s", status, fmt.Sprintf("match based on drive status [%s]", strings.Join(directcsi.SupportedStatusSelectorValues(), ", ")))
}

func setAccessTier(ctx context.Context, accessTier directcsi.AccessTier) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	return processFilteredDrives(
		ctx,
		nil,
		func(drive *directcsi.DirectCSIDrive) bool {
			return drive.Status.DriveStatus != directcsi.DriveStatusUnavailable
		},
		func(drive *directcsi.DirectCSIDrive) error {
			setDriveAccessTier(drive, accessTier)
			return nil
		},
		defaultDriveUpdateFunc(),
		SetAcessTier,
	)
}
