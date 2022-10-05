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
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
)

var (
	accessTierArgs      []string
	accessTierSelectors []types.LabelValue
)

var drivesCmd = &cobra.Command{
	Use:   "drives",
	Short: fmt.Sprintf("Manage %s drives", consts.AppPrettyName),
	Aliases: []string{
		"drive",
		"dr",
	},
}

func init() {
	drivesCmd.AddCommand(listDrivesCmd)
	drivesCmd.AddCommand(formatDrivesCmd)
	drivesCmd.AddCommand(accessTierCmd)
	drivesCmd.AddCommand(releaseDrivesCmd)
}

func validateDriveSelectors() (err error) {
	if driveSelectors, err = getDriveSelectors(); err != nil {
		return err
	}
	if nodeSelectors, err = getNodeSelectors(); err != nil {
		return err
	}
	accessTierSelectors, err = getAccessTierSelectors()

	return err
}

func getAccessTierSelectors() ([]types.LabelValue, error) {
	accessTierSet, err := directpvtypes.StringsToAccessTiers(accessTierArgs...)
	if err != nil {
		return nil, err
	}

	var labelValues []types.LabelValue
	for _, accessTier := range accessTierSet {
		labelValues = append(labelValues, types.NewLabelValue(string(accessTier)))
	}

	return labelValues, nil
}
