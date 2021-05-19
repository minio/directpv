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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog"
)

const XFS = "xfs"

var (
	force     = false
	unrelease = false
)

var formatDrivesCmd = &cobra.Command{
	Use:   "format",
	Short: "format drives in the DirectCSI cluster",
	Long:  "",
	Example: `
# Format all available drives in the cluster
$ kubectl direct-csi drives format --all

# Format all nvme drives in all nodes 
$ kubectl direct-csi drives format --drives '/dev/nvme*'

# Format all drives from a particular node
$ kubectl direct-csi drives format --nodes=directcsi-1

# Format all drives based on the access-tier set [hot|cold|warm]
$ kubectl direct-csi drives format --access-tier=hot

# Format and unrelease all 'released' drives
$ kubectl direct-csi drives format --unrelease --all

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives format --nodes=directcsi-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl direct-csi drives format --nodes=directcsi-1,othernode-2 --status=available
`,
	RunE: func(c *cobra.Command, args []string) error {
		return formatDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glog selector for drive paths")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	formatDrivesCmd.PersistentFlags().BoolVarP(&unrelease, "unrelease", "u", unrelease, "unrelease drives")
	formatDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "format all available drives")

	formatDrivesCmd.PersistentFlags().BoolVarP(&force, "force", "f", force, "force format a drive even if a FS is already present")

	formatDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "format based on access-tier set. The possible values are [hot,cold,warm] ")
}

func formatDrives(ctx context.Context, args []string) error {
	dryRun := viper.GetBool(dryRunFlagName)

	if !all {
		if len(drives) == 0 && len(nodes) == 0 && len(accessTiers) == 0 {
			return fmt.Errorf("atleast one of '%s', '%s' or '%s' should be specified", utils.Bold("--all"), utils.Bold("--drives"), utils.Bold("--nodes"))
		}
	}

	utils.Init()

	directClient := utils.GetDirectCSIClient()
	driveList, err := directClient.DirectCSIDrives().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(driveList.Items) == 0 {
		klog.Errorf("No resource of %s found\n", bold("DirectCSIDrive"))
		return fmt.Errorf("No resources found")
	}

	accessTierSet, aErr := getAccessTierSet(accessTiers)
	if aErr != nil {
		return aErr
	}
	filterDrives := []directcsi.DirectCSIDrive{}
	for _, d := range driveList.Items {
		if d.MatchGlob(nodes, drives, status) {
			if d.MatchAccessTier(accessTierSet) {
				filterDrives = append(filterDrives, d)
			}
		}
	}

	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		dr = strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
		return strings.ReplaceAll(dr, "-part-", "")
	}

	for _, d := range filterDrives {
		if d.Status.DriveStatus == directcsi.DriveStatusUnavailable {
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusInUse {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			klog.Errorf("%s in use. Cannot be formatted", utils.Bold(driveAddr))
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusReady {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			klog.Errorf("%s already owned and managed. Use %s to overwrite", utils.Bold(driveAddr), utils.Bold("--force"))
			continue
		}
		if d.Status.Filesystem != "" && !force {
			driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
			klog.Errorf("%s already has a fs. Use %s to overwrite", utils.Bold(driveAddr), utils.Bold("--force"))
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusReleased {
			if !unrelease {
				driveAddr := fmt.Sprintf("%s:/dev/%s", d.Status.NodeName, driveName(d.Status.Path))
				klog.Errorf("%s is in 'released' state. Use %s to overwrite and make it 'ready'", utils.Bold(driveAddr), utils.Bold("--unrelease"))
				continue
			}
			d.Status.DriveStatus = directcsi.DriveStatusAvailable
		}

		d.Spec.DirectCSIOwned = true
		d.Spec.RequestedFormat = &directcsi.RequestedFormat{
			Filesystem: XFS,
			Force:      force,
		}
		if dryRun {
			if err := utils.LogYAML(d); err != nil {
				return err
			}
			continue
		}
		if _, err := directClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}
