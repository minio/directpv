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
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var errInvalidLabel = errors.New("invalid label")

var driveStatusValues = []string{
	strings.ToLower(string(types.DriveStatusError)),
	strings.ToLower(string(types.DriveStatusLost)),
	strings.ToLower(string(types.DriveStatusMoving)),
	strings.ToLower(string(types.DriveStatusReady)),
	strings.ToLower(string(types.DriveStatusRemoved)),
}

var volumeStatusValues = []string{
	strings.ToLower(string(types.VolumeStatusPending)),
	strings.ToLower(string(types.VolumeStatusReady)),
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
	volumeStatusArgs []string // --status flag of volumes
	pvcFlag          bool     // --pvc flag
	dryRunFlag       bool     // --dry-run flag
	idArgs           []string // --id flag
	showLabels       bool     // --show-labels flag
	labelArgs        []string // --labels flag
	dangerousFlag    bool     // --dangerous flag
	cpuRequest       string   // --cpu-request flag
	cpuLimit         string   // --cpu-limit flag
	memoryRequest    string   // --memory-request flag
	memoryLimit      string   // --memory-limit flag
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

func addResourceFlags(cmd *cobra.Command, usage string) {
	cmd.PersistentFlags().StringVar(&cpuRequest, "cpu-request", cpuRequest, "CPU resource request "+usage)
	cmd.PersistentFlags().StringVar(&cpuLimit, "cpu-limit", cpuLimit, "CPU resource limit "+usage)
	cmd.PersistentFlags().StringVar(&memoryRequest, "memory-request", memoryRequest, "Memory resource request "+usage)
	cmd.PersistentFlags().StringVar(&memoryLimit, "memory-limit", memoryLimit, "Memory resource limit "+usage)
}

func parseResourceRequirements() (resources corev1.ResourceRequirements, err error) {
	if cpuRequest != "" {
		quantity, err := resource.ParseQuantity(cpuRequest)
		if err != nil {
			return resources, fmt.Errorf("invalid value %v for '--cpu-request'; %w", cpuRequest, err)
		}
		if quantity.Sign() < 0 {
			return resources, fmt.Errorf("invalid value %v for '--cpu-request'; must not be negative", cpuRequest)
		}
		if resources.Requests == nil {
			resources.Requests = corev1.ResourceList{}
		}
		resources.Requests[corev1.ResourceCPU] = quantity
	}
	if cpuLimit != "" {
		quantity, err := resource.ParseQuantity(cpuLimit)
		if err != nil {
			return resources, fmt.Errorf("invalid value %v for '--cpu-limit'; %w", cpuLimit, err)
		}
		if quantity.Sign() < 0 {
			return resources, fmt.Errorf("invalid value %v for '--cpu-limit'; must not be negative", cpuLimit)
		}
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
		resources.Limits[corev1.ResourceCPU] = quantity
	}
	if memoryRequest != "" {
		quantity, err := resource.ParseQuantity(memoryRequest)
		if err != nil {
			return resources, fmt.Errorf("invalid value %v for '--memory-request'; %w", memoryRequest, err)
		}
		if quantity.Sign() < 0 {
			return resources, fmt.Errorf("invalid value %v for '--memory-request'; must not be negative", memoryRequest)
		}
		if resources.Requests == nil {
			resources.Requests = corev1.ResourceList{}
		}
		resources.Requests[corev1.ResourceMemory] = quantity
	}
	if memoryLimit != "" {
		quantity, err := resource.ParseQuantity(memoryLimit)
		if err != nil {
			return resources, fmt.Errorf("invalid value %v for '--memory-limit'; %w", memoryLimit, err)
		}
		if quantity.Sign() < 0 {
			return resources, fmt.Errorf("invalid value %v for '--memory-limit'; must not be negative", memoryLimit)
		}
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
		resources.Limits[corev1.ResourceMemory] = quantity
	}

	if cpuRequest != "" && cpuLimit != "" {
		request := resources.Requests[corev1.ResourceCPU]
		limit := resources.Limits[corev1.ResourceCPU]
		if request.Cmp(limit) > 0 {
			return resources, fmt.Errorf("invalid value %v for '--cpu-request'; must be less than or equal to '--cpu-limit' value %v", cpuRequest, cpuLimit)
		}
	}
	if memoryRequest != "" && memoryLimit != "" {
		request := resources.Requests[corev1.ResourceMemory]
		limit := resources.Limits[corev1.ResourceMemory]
		if request.Cmp(limit) > 0 {
			return resources, fmt.Errorf("invalid value %v for '--memory-request'; must be less than or equal to '--memory-limit' value %v", memoryRequest, memoryLimit)
		}
	}

	return resources, nil
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

	driveStatusSelectors  []types.DriveStatus
	driveIDSelectors      []types.DriveID
	volumeStatusSelectors []types.VolumeStatus
	labelSelectors        map[types.LabelKey]types.LabelValue

	dryRunPrinter func(interface{})
)

func validateNodeArgs() error {
	var values []string

	for i := range nodesArgs {
		nodesArgs[i] = strings.TrimSpace(nodesArgs[i])
		if nodesArgs[i] == "" {
			return errors.New("empty node name")
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
			return errors.New("empty drive name")
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
		status, err := types.ToDriveStatus(driveStatusArgs[i])
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
			return errors.New("empty drive ID")
		}
		if !utils.IsUUID(driveIDArgs[i]) {
			return fmt.Errorf("invalid drive ID %v", driveIDArgs[i])
		}
		driveIDSelectors = append(driveIDSelectors, types.DriveID(driveIDArgs[i]))
	}
	return nil
}

func validatePodNameArgs() error {
	var values []string

	for i := range podNameArgs {
		podNameArgs[i] = strings.TrimSpace(podNameArgs[i])
		if podNameArgs[i] == "" {
			return errors.New("empty pod name")
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
			return errors.New("empty pod namespace")
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
			return errors.New("empty volume name")
		}
	}
	return nil
}

func validateVolumeStatusArgs() error {
	for i := range volumeStatusArgs {
		volumeStatusArgs[i] = strings.TrimSpace(volumeStatusArgs[i])
		status, err := types.ToVolumeStatus(volumeStatusArgs[i])
		if err != nil {
			return err
		}
		volumeStatusSelectors = append(volumeStatusSelectors, status)
	}
	return nil
}

func validateLabelArgs() error {
	if labelSelectors == nil {
		labelSelectors = make(map[types.LabelKey]types.LabelValue)
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
