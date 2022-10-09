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
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	accessTierArgs  []string
	driveStatusArgs []string

	accessTierSelectors []directpvtypes.LabelValue
)

func getAccessTierSelectors() ([]directpvtypes.LabelValue, error) {
	accessTierSet, err := directpvtypes.StringsToAccessTiers(accessTierArgs...)
	if err != nil {
		return nil, err
	}

	var labelValues []directpvtypes.LabelValue
	for _, accessTier := range accessTierSet {
		labelValues = append(labelValues, directpvtypes.NewLabelValue(string(accessTier)))
	}

	return labelValues, nil
}

func getDriveStatusSelectors() error {
	statusValues := driveStatusValues()
	var invalidValues []string
	for i, status := range driveStatusArgs {
		if !utils.Contains(statusValues, strings.ToLower(status)) {
			invalidValues = append(invalidValues, status)
		}
		driveStatusArgs[i] = strings.ToLower(status)
	}

	if len(invalidValues) != 0 {
		return fmt.Errorf("unknown drive status %v", invalidValues)
	}

	return nil
}

func validateDriveSelectors() (err error) {
	if driveSelectors, err = getDriveSelectors(); err != nil {
		return err
	}
	if nodeSelectors, err = getNodeSelectors(); err != nil {
		return err
	}
	if accessTierSelectors, err = getAccessTierSelectors(); err != nil {
		return err
	}

	return getDriveStatusSelectors()
}

var drivesCmd = &cobra.Command{
	Use:     "drives",
	Aliases: []string{"drive", "dr"},
	Short:   "Manage drives.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}
		return validateDriveSelectors()
	},
}

func driveStatusValues() []string {
	return []string{
		strings.ToLower(string(directpvtypes.DriveStatusReady)),
		strings.ToLower(string(directpvtypes.DriveStatusLost)),
		strings.ToLower(string(directpvtypes.DriveStatusError)),
		strings.ToLower(string(directpvtypes.DriveStatusReleased)),
	}
}

func accessTierValues() []string {
	return []string{
		strings.ToLower(string(directpvtypes.AccessTierDefault)),
		strings.ToLower(string(directpvtypes.AccessTierWarm)),
		strings.ToLower(string(directpvtypes.AccessTierHot)),
		strings.ToLower(string(directpvtypes.AccessTierCold)),
	}
}

func init() {
	drivesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter output by nodes optionally in ellipses pattern.")
	drivesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter output by drives optionally in ellipses pattern.")
	drivesCmd.PersistentFlags().StringSliceVarP(&accessTierArgs, "access-tier", "", accessTierArgs, fmt.Sprintf("Filter output by access tier. One of: %v", strings.Join(accessTierValues(), "|")))
	drivesCmd.PersistentFlags().StringSliceVarP(&driveStatusArgs, "status", "", driveStatusArgs, fmt.Sprintf("Filter output by drive status. One of: %v", strings.Join(driveStatusValues(), "|")))

	drivesCmd.AddCommand(drivesListCmd)
	drivesCmd.AddCommand(drivesFormatCmd)
	drivesCmd.AddCommand(drivesSetCmd)
	drivesCmd.AddCommand(drivesReleaseCmd)
	drivesCmd.AddCommand(drivesCordonCmd)
	drivesCmd.AddCommand(drivesUncordonCmd)
	drivesCmd.AddCommand(drivesMoveCmd)
}
