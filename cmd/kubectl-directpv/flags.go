// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var driveStatusValues = []string{
	strings.ToLower(string(directpvtypes.DriveStatusError)),
	strings.ToLower(string(directpvtypes.DriveStatusLost)),
	strings.ToLower(string(directpvtypes.DriveStatusMoving)),
	strings.ToLower(string(directpvtypes.DriveStatusReady)),
	strings.ToLower(string(directpvtypes.DriveStatusReleased)),
}

var accessTierValues = []string{
	strings.ToLower(string(directpvtypes.AccessTierCold)),
	strings.ToLower(string(directpvtypes.AccessTierDefault)),
	strings.ToLower(string(directpvtypes.AccessTierHot)),
	strings.ToLower(string(directpvtypes.AccessTierWarm)),
}

var volumeStatusValues = []string{
	strings.ToLower(string(directpvtypes.VolumeStatusPending)),
	strings.ToLower(string(directpvtypes.VolumeStatusReady)),
}

var (
	configDir        = getDefaultConfigDir() // --config-dir flag
	kubeconfig       string                  // --kubeconfig flag
	quietFlag        bool                    // --quiet flag
	outputFormat     string                  // --output flag
	noHeaders        bool                    // --no-headers flag
	allFlag          bool                    // --all flag
	nodeArgs         []string                // --node flag
	driveNameArgs    []string                // --drive-name flag
	accessTierArgs   []string                // --access-tier flag
	driveStatusArgs  []string                // --status flag of drives
	driveIDArgs      []string                // --drive flag
	podNameArgs      []string                // --pod-name flag
	podNSArgs        []string                // --pod-namespace flag
	volumeStatusArgs []string                // --status flag of volumes
	pvcFlag          bool                    // --pvc flag
	dryRunFlag       bool                    // --dry-run flag
)

func addAllFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().BoolVar(&allFlag, "all", allFlag, usage)
}

func addDryRunFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", dryRunFlag, "Run in dry-run mode")
}

func addNodeFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, usage+"; supports ellipses pattern e.g. node{1...10}")
}

func addDriveNameFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVarP(&driveNameArgs, "drive-name", "d", driveNameArgs, usage+"; supports ellipses pattern e.g. sd{a...z}")
}

func addAccessTierFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&accessTierArgs, "access-tier", accessTierArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(accessTierValues, "|")))
}

func addDriveStatusFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&driveStatusArgs, "status", driveStatusArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(driveStatusValues, "|")))
}

func addDriveIDFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&driveIDArgs, "drive-id", driveIDArgs, usage)
}

func addPodNameFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&podNameArgs, "pod-name", podNameArgs, usage+"; supports ellipses pattern e.g. minio-{0...4}")
}

func addPodNSFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&podNSArgs, "pod-namespace", podNameArgs, usage+"; supports ellipses pattern e.g. tenant-{0...3}")
}

func addVolumeStatusFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&volumeStatusArgs, "status", volumeStatusArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(volumeStatusValues, "|")))
}

var (
	wideOutput bool
	jsonOutput bool
	yamlOutput bool

	driveStatusSelectors  []directpvtypes.DriveStatus
	driveIDSelectors      []directpvtypes.DriveID
	volumeStatusSelectors []directpvtypes.VolumeStatus

	printer func(interface{}) error
)

func validateNodeArgs() error {
	var values []string

	for i := range nodeArgs {
		nodeArgs[i] = strings.TrimSpace(nodeArgs[i])
		if nodeArgs[i] == "" {
			return fmt.Errorf("empty node name")
		}
		result, err := ellipsis.Expand(nodeArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	nodeArgs = values
	return nil
}

func validateDriveNameArgs() error {
	var values []string

	for i := range driveNameArgs {
		driveNameArgs[i] = strings.TrimSpace(utils.TrimDevPrefix(driveNameArgs[i]))
		if driveNameArgs[i] == "" {
			return fmt.Errorf("empty drive name")
		}
		result, err := ellipsis.Expand(driveNameArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	driveNameArgs = values
	return nil
}

func validateAccessTierArgs() error {
	for i := range accessTierArgs {
		accessTierArgs[i] = strings.TrimSpace(accessTierArgs[i])
		if !utils.Contains(accessTierValues, accessTierArgs[i]) {
			return fmt.Errorf("unknown access-tier %v", accessTierArgs[i])
		}
	}

	accessTiers, err := directpvtypes.StringsToAccessTiers(accessTierArgs...)
	if err != nil {
		return err
	}

	for i := range accessTiers {
		accessTierArgs[i] = string(accessTiers[i])
	}

	return nil
}

func validateDriveStatusArgs() error {
	for i := range driveStatusArgs {
		driveStatusArgs[i] = strings.TrimSpace(driveStatusArgs[i])
		status, err := directpvtypes.ToDriveStatus(driveStatusArgs[i])
		if err != nil {
			return err
		}
		driveStatusSelectors = append(driveStatusSelectors, status)
	}
	return nil
}

func validateDriveIDArgs() error {
	for i := range driveIDArgs {
		driveIDArgs[i] = strings.TrimSpace(driveIDArgs[i])
		if driveIDArgs[i] == "" {
			return fmt.Errorf("empty drive ID")
		}
		driveIDSelectors = append(driveIDSelectors, directpvtypes.DriveID(driveIDArgs[i]))
	}
	return nil
}

func validatePodNameArgs() error {
	var values []string

	for i := range podNameArgs {
		podNameArgs[i] = strings.TrimSpace(podNameArgs[i])
		if podNameArgs[i] == "" {
			return fmt.Errorf("empty pod name")
		}
		result, err := ellipsis.Expand(podNameArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	podNameArgs = values
	return nil
}

func validatePodNSArgs() error {
	var values []string

	for i := range podNSArgs {
		podNSArgs[i] = strings.TrimSpace(podNSArgs[i])
		if podNSArgs[i] == "" {
			return fmt.Errorf("empty pod namespace")
		}
		result, err := ellipsis.Expand(podNSArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	podNSArgs = values
	return nil
}

func validateVolumeNameArgs() error {
	for i := range volumeNameArgs {
		volumeNameArgs[i] = strings.TrimSpace(volumeNameArgs[i])
		if volumeNameArgs[i] == "" {
			return fmt.Errorf("empty volume name")
		}
	}
	return nil
}

func validateVolumeStatusArgs() error {
	for i := range volumeStatusArgs {
		volumeStatusArgs[i] = strings.TrimSpace(volumeStatusArgs[i])
		status, err := directpvtypes.ToVolumeStatus(volumeStatusArgs[i])
		if err != nil {
			return err
		}
		volumeStatusSelectors = append(volumeStatusSelectors, status)
	}
	return nil
}
