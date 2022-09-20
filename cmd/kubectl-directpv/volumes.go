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
	"errors"
	"fmt"
	"regexp"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	volumeStatusArgs []string
	podNameArgs      []string
	podNSArgs        []string

	volumeStatusSelectors []string
	podNameSelectors      []types.LabelValue
	podNSSelectors        []types.LabelValue
)

var volumesCmd = &cobra.Command{
	Use:   "volumes",
	Short: "Manage DirectPV Volumes",
	Aliases: []string{
		"volume",
		"vol",
	},
}

func init() {
	volumesCmd.AddCommand(listVolumesCmd)
	volumesCmd.AddCommand(purgeVolumesCmd)
}

var (
	globRegexp                = regexp.MustCompile(`(^|[^\\])[\*\?\[]`)
	errGlobPatternUnsupported = errors.New("glob patterns are unsupported")
)

func getSelectorValues(selectors []string) (values []types.LabelValue, err error) {
	for _, selector := range selectors {
		if globRegexp.MatchString(selector) {
			return nil, errGlobPatternUnsupported
		}

		result, err := ellipsis.Expand(selector)
		if err != nil {
			return nil, err
		}

		for _, value := range result {
			values = append(values, types.NewLabelValue(value))
		}
	}

	return values, nil
}

func getDriveSelectors() ([]types.LabelValue, error) {
	var values []string
	for i := range driveArgs {
		if utils.TrimDevPrefix(driveArgs[i]) == "" {
			return nil, fmt.Errorf("empty device name %v", driveArgs[i])
		}
		values = append(values, utils.TrimDevPrefix(driveArgs[i]))
	}
	return getSelectorValues(values)
}

func getNodeSelectors() ([]types.LabelValue, error) {
	for i := range nodeArgs {
		if utils.TrimDevPrefix(nodeArgs[i]) == "" {
			return nil, fmt.Errorf("empty node name %v", nodeArgs[i])
		}
	}
	return getSelectorValues(nodeArgs)
}

func getPodNameSelectors() ([]types.LabelValue, error) {
	for i := range podNameArgs {
		if utils.TrimDevPrefix(podNameArgs[i]) == "" {
			return nil, fmt.Errorf("empty pod name %v", podNameArgs[i])
		}
	}
	return getSelectorValues(podNameArgs)
}

func getPodNamespaceSelectors() ([]types.LabelValue, error) {
	for i := range podNSArgs {
		if utils.TrimDevPrefix(podNSArgs[i]) == "" {
			return nil, fmt.Errorf("empty pod namespace %v", podNSArgs[i])
		}
	}
	return getSelectorValues(podNSArgs)
}

func getVolumeStatusSelectors() ([]string, error) {
	for _, status := range volumeStatusArgs {
		switch directpvtypes.VolumeConditionType(strings.Title(status)) {
		case directpvtypes.VolumeConditionTypePublished:
		case directpvtypes.VolumeConditionTypeStaged:
		case directpvtypes.VolumeConditionTypeReady:
		default:
			return nil, fmt.Errorf("unknown volume condition type %v", status)
		}
	}
	return volumeStatusArgs, nil
}

func validateVolumeSelectors() (err error) {
	if driveSelectors, err = getDriveSelectors(); err != nil {
		return err
	}

	if nodeSelectors, err = getNodeSelectors(); err != nil {
		return err
	}

	if volumeStatusSelectors, err = getVolumeStatusSelectors(); err != nil {
		return err
	}

	if podNameSelectors, err = getPodNameSelectors(); err != nil {
		return err
	}

	podNSSelectors, err = getPodNamespaceSelectors()

	return err
}
