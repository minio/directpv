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
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/spf13/cobra"

	"k8s.io/klog/v2"
)

var releaseDrivesCmd = &cobra.Command{
	Use:   "release",
	Short: "release drives from the DirectCSI cluster",
	Long:  "",
	Example: `
 # Release all drives in the cluster
 $ kubectl direct-csi drives release --all
 
 # Release the 'sdf' drives in all nodes
 $ kubectl direct-csi drives release --drives '/dev/sdf'

 # Release the selective drives using ellipses notation for drive paths
 $ kubectl direct-csi drives release --drives '/dev/sd{a...z}'
 
 # Release the drives from selective nodes using ellipses notation for node names
 $ kubectl direct-csi drives release --nodes 'directcsi-{1...3}'
 
 # Release all drives from a particular node
 $ kubectl direct-csi drives release --nodes=directcsi-1
 
 # Release all drives based on the access-tier set [hot|cold|warm]
 $ kubectl direct-csi drives release --access-tier=hot
 
 # Combine multiple parameters using multi-arg
 $ kubectl direct-csi drives release --nodes=directcsi-1 --nodes=othernode-2 --status=available
 
 # Combine multiple parameters using csv
 $ kubectl direct-csi drives release --nodes=directcsi-1,othernode-2 --status=ready
 `,
	RunE: func(c *cobra.Command, args []string) error {
		if !all {
			if len(drives) == 0 && len(nodes) == 0 && len(accessTiers) == 0 && len(args) == 0 {
				return fmt.Errorf("atleast one among ['%s','%s','%s','%s'] should be specified", utils.Bold("--all"), utils.Bold("--drives"), utils.Bold("--nodes"), utils.Bold("--access-tier"))
			}
		}
		if err := validateDriveSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}

		return releaseDrives(c.Context(), args)
	},
	Aliases: []string{},
}

func init() {
	releaseDrivesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	releaseDrivesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	releaseDrivesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "release all available drives")

	releaseDrivesCmd.PersistentFlags().StringSliceVarP(&accessTiers, "access-tier", "", accessTiers, "release based on access-tier set. The possible values are [hot,cold,warm] ")
}

func releaseDrives(ctx context.Context, IDArgs []string) error {
	driveName := func(val string) string {
		dr := strings.ReplaceAll(val, sys.DirectCSIDevRoot+"/", "")
		dr = strings.ReplaceAll(dr, sys.HostDevRoot+"/", "")
		return strings.ReplaceAll(dr, "-part-", "")
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	directCSIClient := client.GetDirectCSIClient()
	return processFilteredDrives(
		ctx,
		directCSIClient.DirectCSIDrives(),
		IDArgs,
		func(drive *directcsi.DirectCSIDrive) bool {
			if drive.Status.DriveStatus == directcsi.DriveStatusUnavailable {
				return false
			}

			if drive.Status.DriveStatus == directcsi.DriveStatusInUse {
				driveAddr := fmt.Sprintf("%s:/dev/%s", drive.Status.NodeName, driveName(drive.Status.Path))
				klog.Errorf("%s in use. Cannot be released", utils.Bold(driveAddr))
				return false
			}

			if drive.Status.DriveStatus == directcsi.DriveStatusReleased {
				driveAddr := fmt.Sprintf("%s:/dev/%s", drive.Status.NodeName, driveName(drive.Status.Path))
				klog.Errorf("%s already in 'released' state", utils.Bold(driveAddr))
				return false
			}

			if drive.Status.DriveStatus != directcsi.DriveStatusReady {
				driveAddr := fmt.Sprintf("%s:/dev/%s", drive.Status.NodeName, driveName(drive.Status.Path))
				klog.Errorf("%s in '%s' state. only 'ready' drives can be released", utils.Bold(driveAddr), string(drive.Status.DriveStatus))
				return false
			}
			return true
		},
		func(drive *directcsi.DirectCSIDrive) error {
			drive.Status.DriveStatus = directcsi.DriveStatusReleased
			drive.Spec.DirectCSIOwned = false
			drive.Spec.RequestedFormat = nil
			return nil
		},
		defaultDriveUpdateFunc(directCSIClient),
		DriveRelease,
	)
}
