/*
 * This file is part of MinIO Direct PV
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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var volumeStatus, volumeStatusList, podNames, podNameGlobs, podNss, podNsGlobs []string
var podNameSelectorValues, podNsSelectorValues []utils.LabelValue

var volumesCmd = &cobra.Command{
	Use:   "volumes",
	Short: "Manage Direct Persistent Volumes",
	Long:  "",
	Aliases: []string{
		"volume",
		"vol",
	},
}

func validateVolumeSelectors() (err error) {
	driveGlobs, driveSelectorValues, err = getValidDriveSelectors(drives)
	if err != nil {
		return err
	}

	nodeGlobs, nodeSelectorValues, err = getValidNodeSelectors(nodes)
	if err != nil {
		return err
	}

	podNameGlobs, podNameSelectorValues, err = getValidPodNameSelectors(podNames)
	if err != nil {
		return err
	}

	podNsGlobs, podNsSelectorValues, err = getValidPodNameSpaceSelectors(podNss)
	if err != nil {
		return err
	}

	volumeStatusList, err = getValidVolumeStatusSelectors(volumeStatus)

	return err
}

func init() {
	volumesCmd.AddCommand(listVolumesCmd)
}
