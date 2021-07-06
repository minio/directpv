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
	"sync"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

const XFS = "xfs"

var (
	force = false
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

# Combine multiple parameters using multi-arg
$ kubectl direct-csi drives format --nodes=directcsi-1 --nodes=othernode-2 --status=available

# Combine multiple parameters using csv
$ kubectl direct-csi drives format --nodes=directcsi-1,othernode-2 --status=available

# Format a drive by it's drive-id
$ kubectl direct-csi drives format <drive_id>

# Format more than one drive by their drive-ids
$ kubectl direct-csi drives format <drive_id_1> <drive_id_2>
`,
	RunE: func(c *cobra.Command, args []string) error {
		return formatDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "glog selector for drive paths")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "glob selector for node names")
	formatDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "format all available drives")
	formatDrivesCmd.PersistentFlags().BoolVarP(&force, "force", "f", force, "force format a drive even if a FS is already present")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers,
		"format based on access-tier set. The possible values are hot|cold|warm")
}

func formatDrives(ctx context.Context, args []string) error {
	if !all {
		if len(drives) == 0 && len(nodes) == 0 && len(accessTiers) == 0 && len(args) == 0 {
			return fmt.Errorf("atleast one of '%s', '%s' or '%s' should be specified",
				utils.Bold("--all"),
				utils.Bold("--drives"),
				utils.Bold("--nodes"))
		}
	}

	directClient := utils.GetDirectCSIClient()

	var driveCh <-chan directcsi.DirectCSIDrive
	if len(args) > 0 {
		driveCh = getDrivesByIds(ctx, args)
	} else {
		driveCh = getDrives(ctx, nodes, drives, accessTiers)
	}

	wg := sync.WaitGroup{}
	accessTierSet, aErr := getAccessTierSet(accessTiers)
	if aErr != nil {
		return aErr
	}
	for d := range driveCh {
		if !d.MatchGlob(nodes, drives, status) {
			continue
		}

		if !d.MatchAccessTier(accessTierSet) {
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusUnavailable {
			continue
		}

		path := canonicalNameFromPath(d.Status.Path)
		nodeName := d.Status.NodeName
		driveAddr := fmt.Sprintf("%s:/dev/%s", nodeName, path)

		if d.Status.DriveStatus == directcsi.DriveStatusInUse {
			klog.Errorf("%s is in use. Cannot be formatted",
				utils.Bold(driveAddr))
			continue
		}

		if d.Status.DriveStatus == directcsi.DriveStatusReady {
			klog.Errorf("%s is already owned and managed",
				utils.Bold(driveAddr))
			continue
		}
		if d.Status.Filesystem != "" && !force {
			klog.Errorf("%s already has a fs. Use %s to overwrite",
				utils.Bold(driveAddr), utils.Bold("--force"))
			continue
		}
		if d.Status.DriveStatus == directcsi.DriveStatusReleased {
			klog.Errorf("%s is in 'released' state. Use 'kubectl direct-csi drives unrelease --drive %s --nodes %s' before formatting",
				utils.Bold(driveAddr), path, nodeName)
			continue
		}

		d.Spec.DirectCSIOwned = true
		d.Spec.RequestedFormat = &directcsi.RequestedFormat{
			Filesystem: XFS,
			Force:      force,
		}
		if dryRun {
			if err := printer(d); err != nil {
				klog.ErrorS(err, "error marshaling drives", "format", outputMode)
			}
		} else {
			threadiness <- struct{}{}
			wg.Add(1)
			go func(d directcsi.DirectCSIDrive) {
				defer func() {
					wg.Done()
					<-threadiness
				}()

				if _, err := directClient.DirectCSIDrives().Update(ctx, &d, metav1.UpdateOptions{}); err != nil {
					klog.ErrorS(err, "failed to format drive", "drive", driveAddr)
				}
			}(d)
		}
	}
	wg.Wait()

	return nil
}
