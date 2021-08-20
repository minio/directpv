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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

# Unsets the 'access-tier' tag on all the drives from a particular node
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1 --nodes=othernode-2 --status=ready

# Combine multiple parameters using csv
$ kubectl direct-csi drives access-tier unset --nodes=directcsi-1,othernode-2 --access-tier=hot
`,
	RunE: func(c *cobra.Command, args []string) error {
		return unsetAccessTier(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	accessTierUnset.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glob selector for drive paths")
	accessTierUnset.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	accessTierUnset.PersistentFlags().BoolVarP(&all, "all", "a", all, "untag all available drives")
	accessTierUnset.PersistentFlags().StringSliceVarP(&status, "status", "s", status, "glob prefix match for drive status")
	accessTierUnset.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "format based on access-tier set. The possible values are [hot,cold,warm] ")
}

func unsetAccessTier(ctx context.Context, args []string) error {
	if !all {
		if len(drives) == 0 && len(nodes) == 0 && len(status) == 0 && len(accessTiers) == 0 {
			return fmt.Errorf("atleast one of '%s', '%s', '%s', '%s', or '%s' should be specified",
				utils.Bold("--all"),
				utils.Bold("--drives"),
				utils.Bold("--nodes"),
				utils.Bold("--status"),
				utils.Bold("--access-tier"))
		}
	}

	directClient := utils.GetDirectCSIClient()
	driveList, err := utils.GetDriveList(ctx, directClient.DirectCSIDrives(), nil, nil, nil)
	if err != nil {
		return err
	}

	if len(driveList) == 0 {
		klog.Errorf("No resource of %s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	accessTierSet, aErr := getAccessTierSet(accessTiers)
	if aErr != nil {
		return aErr
	}
	filterDrives := []directcsi.DirectCSIDrive{}
	for _, d := range driveList {
		if all {
			filterDrives = append(filterDrives, d)
			continue
		}
		if d.MatchGlob(nodes, drives, status) {
			if d.MatchAccessTier(accessTierSet) {
				filterDrives = append(filterDrives, d)
			}
		}
	}

	for _, d := range filterDrives {
		d.Status.AccessTier = directcsi.AccessTierUnknown
		utils.SetAccessTierLabel(&d, directcsi.AccessTierUnknown)

		if dryRun {
			if err := printer(d); err != nil {
				klog.ErrorS(err, "error marshaling drives", "format", outputMode)
			}
			continue
		}

		if _, err := directClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
