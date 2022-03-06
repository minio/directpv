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
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var status, accessTiers, statusGlobs []string
var accessTierSelectorValues []utils.LabelValue
var driveStatusList []directcsi.DriveStatus

var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: utils.BinaryNameTransform("Manage Drives in {{ . }} cluster"),
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
