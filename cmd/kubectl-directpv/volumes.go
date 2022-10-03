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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	podNameArgs []string
	podNSArgs   []string

	stagedFlag       bool
	podNameSelectors []types.LabelValue
	podNSSelectors   []types.LabelValue
)

var volumesCmd = &cobra.Command{
	Use:   "volumes",
	Short: fmt.Sprintf("Manage %s Volumes", consts.AppPrettyName),
	Aliases: []string{
		"volume",
		"vol",
	},
}

func init() {
	volumesCmd.AddCommand(listVolumesCmd)
	volumesCmd.AddCommand(purgeVolumesCmd)
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

func validateVolumeSelectors() (err error) {
	if driveSelectors, err = getDriveSelectors(); err != nil {
		return err
	}

	if nodeSelectors, err = getNodeSelectors(); err != nil {
		return err
	}

	if podNameSelectors, err = getPodNameSelectors(); err != nil {
		return err
	}

	podNSSelectors, err = getPodNamespaceSelectors()

	return err
}
