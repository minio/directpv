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
	"errors"
	"fmt"
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var errInvalidLabel = errors.New("invalid label")

var driveStatusValues = []string{
	strings.ToLower(string(directpvtypes.DriveStatusError)),
	strings.ToLower(string(directpvtypes.DriveStatusLost)),
	strings.ToLower(string(directpvtypes.DriveStatusMoving)),
	strings.ToLower(string(directpvtypes.DriveStatusReady)),
	strings.ToLower(string(directpvtypes.DriveStatusRemoved)),
}

var volumeStatusValues = []string{
	strings.ToLower(string(directpvtypes.VolumeStatusPending)),
	strings.ToLower(string(directpvtypes.VolumeStatusReady)),
}

var jobStatusValues = []string{
	string(jobs.JobStatusActive),
	string(jobs.JobStatusFailed),
	string(jobs.JobStatusSucceeded),
}

var jobTypeValues = []string{
	string(jobs.JobTypeCopy),
}

var (
	kubeconfig       string   // --kubeconfig flag
	quietFlag        bool     // --quiet flag
	outputFormat     string   // --output flag
	noHeaders        bool     // --no-headers flag
	allFlag          bool     // --all flag
	nodesArgs        []string // --nodes flag
	drivesArgs       []string // --drives flag
	driveStatusArgs  []string // --status flag of drives
	driveIDArgs      []string // --drive-id flag
	podNameArgs      []string // --pod-name flag
	podNSArgs        []string // --pod-namespace flag
	volumeStatusArgs []string // --status flag for volumes
	jobStatusArgs    []string // --status flag for jobs
	jobTypeArgs      []string // --type flag for jobs
	pvcFlag          bool     // --pvc flag
	dryRunFlag       bool     // --dry-run flag
	idArgs           []string // --id flag
	showLabels       bool     // --show-labels flag
	labelArgs        []string // --labels flag
	dangerousFlag    bool     // --dangerous flag
)

func addAllFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().BoolVar(&allFlag, "all", allFlag, usage)
}

func addDryRunFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", dryRunFlag, usage)
}

func addDangerousFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().BoolVar(&dangerousFlag, "dangerous", dangerousFlag, usage)
}

func addNodesFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVarP(&nodesArgs, "nodes", "n", nodesArgs, usage+"; supports ellipses pattern e.g. node{1...10}")
}

func addDrivesFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVarP(&drivesArgs, "drives", "d", drivesArgs, usage+"; supports ellipses pattern e.g. sd{a...z}")
}

func addOutputFormatFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", outputFormat, usage)
}

func addDriveStatusFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&driveStatusArgs, "status", driveStatusArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(driveStatusValues, "|")))
}

func addDriveIDFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&driveIDArgs, "drive-id", driveIDArgs, usage)
}

func addPodNameFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&podNameArgs, "pod-names", podNameArgs, usage+"; supports ellipses pattern e.g. minio-{0...4}")
}

func addPodNSFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&podNSArgs, "pod-namespaces", podNameArgs, usage+"; supports ellipses pattern e.g. tenant-{0...3}")
}

func addVolumeStatusFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&volumeStatusArgs, "status", volumeStatusArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(volumeStatusValues, "|")))
}

func addJobsStatusFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&jobStatusArgs, "status", jobStatusArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(jobStatusValues, "|")))
}

func addJobsTypeFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&jobTypeArgs, "type", jobTypeArgs, fmt.Sprintf("%v; one of: %v", usage, strings.Join(jobTypeValues, "|")))
}

func addIDFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&idArgs, "ids", idArgs, usage)
}

func addShowLabelsFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&showLabels, "show-labels", showLabels, "show all labels as the last column (default hide labels column)")
}

func addLabelsFlag(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringSliceVar(&labelArgs, "labels", labelArgs, usage+"; supports comma separated kv pairs. e.g. tier=hot,region=east")
}

func setFlagOpts(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.InheritedFlags().SortFlags = false
	cmd.LocalFlags().SortFlags = false
	cmd.LocalNonPersistentFlags().SortFlags = false
	cmd.NonInheritedFlags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
}

var (
	wideOutput bool

	driveStatusSelectors  []directpvtypes.DriveStatus
	driveIDSelectors      []directpvtypes.DriveID
	volumeStatusSelectors []directpvtypes.VolumeStatus
	labelSelectors        map[directpvtypes.LabelKey]directpvtypes.LabelValue
	jobStatusSelectors    []jobs.JobStatus
	jobTypeSelectors      []jobs.JobType

	dryRunPrinter func(interface{})
)

func validateNodeArgs() error {
	var values []string

	for i := range nodesArgs {
		nodesArgs[i] = strings.TrimSpace(nodesArgs[i])
		if nodesArgs[i] == "" {
			return fmt.Errorf("empty node name")
		}
		result, err := ellipsis.Expand(nodesArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	nodesArgs = values
	return nil
}

func validateDriveNameArgs() error {
	var values []string

	for i := range drivesArgs {
		drivesArgs[i] = strings.TrimSpace(utils.TrimDevPrefix(drivesArgs[i]))
		if drivesArgs[i] == "" {
			return fmt.Errorf("empty drive name")
		}
		result, err := ellipsis.Expand(drivesArgs[i])
		if err != nil {
			return err
		}
		values = append(values, result...)
	}

	drivesArgs = values
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
		if !utils.IsUUID(driveIDArgs[i]) {
			return fmt.Errorf("invalid drive ID %v", driveIDArgs[i])
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
	return validateNameArgs(volumeNameArgs)
}

func validateNameArgs(args []string) error {
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
		if args[i] == "" {
			return fmt.Errorf("empty name")
		}
	}
	return nil
}

func validateJobNameArgs() error {
	return validateNameArgs(jobNameArgs)
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

func validateJobStatusArgs() error {
	for i := range jobStatusArgs {
		jobStatusArgs[i] = strings.TrimSpace(jobStatusArgs[i])
		status, err := jobs.ToStatus(jobStatusArgs[i])
		if err != nil {
			return err
		}
		jobStatusSelectors = append(jobStatusSelectors, status)
	}
	return nil
}

func validateJobTypeArgs() error {
	for i := range jobTypeArgs {
		jobTypeArgs[i] = strings.ToLower(strings.TrimSpace(jobTypeArgs[i]))
		jobType, err := jobs.ToType(jobTypeArgs[i])
		if err != nil {
			return err
		}
		jobTypeSelectors = append(jobTypeSelectors, jobType)
	}
	return nil
}

func validateLabelArgs() error {
	if labelSelectors == nil {
		labelSelectors = make(map[directpvtypes.LabelKey]directpvtypes.LabelValue)
	}
	for i := range labelArgs {
		tokens := strings.Split(labelArgs[i], "=")
		if len(tokens) != 2 {
			return errInvalidLabel
		}
		key, err := types.NewLabelKey(tokens[0])
		if err != nil {
			return err
		}
		value, err := types.NewLabelValue(tokens[1])
		if err != nil {
			return err
		}

		labelSelectors[key] = value
	}
	return nil
}
