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
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var status, accessTiers, statusGlobs []string
var accessTierSelectorValues []utils.LabelValue
var driveStatusList []directcsi.DriveStatus

var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: "Manage Drives on DirectCSI",
	Long:  "",
	Aliases: []string{
		"drive",
		"dr",
	},
}

func validateDriveSelectors() (err error) {
	driveGlobs, driveSelectorValues, err = getValidDriveSelectors(drives)
	if err != nil {
		return err
	}
	nodeGlobs, nodeSelectorValues, err = getValidNodeSelectors(nodes)
	if err != nil {
		return err
	}
	accessTierSelectorValues, err = getValidAccessTierSelectors(accessTiers)
	if err != nil {
		return err
	}
	statusGlobs, driveStatusList, err = getValidDriveStatusSelectors(status)

	return err
}

func init() {
	drivesCmd.AddCommand(listDrivesCmd)
	drivesCmd.AddCommand(formatDrivesCmd)
	drivesCmd.AddCommand(drivesAccessTierCmd)
	drivesCmd.AddCommand(releaseDrivesCmd)
	drivesCmd.AddCommand(unreleaseDrivesCmd)
}
