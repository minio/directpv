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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	errEmptyValue = errors.New("empty value provided")
)

var (
	podNameArgs []string
	podNSArgs   []string

	podNameSelectors []directpvtypes.LabelValue
	podNSSelectors   []directpvtypes.LabelValue
)

func getPodNameSelectors() ([]directpvtypes.LabelValue, error) {
	for i := range podNameArgs {
		if utils.TrimDevPrefix(podNameArgs[i]) == "" {
			return nil, fmt.Errorf("empty pod name %v", podNameArgs[i])
		}
	}
	return getSelectorValues(podNameArgs)
}

func getPodNamespaceSelectors() ([]directpvtypes.LabelValue, error) {
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

var volumesCmd = &cobra.Command{
	Use:     "volumes",
	Aliases: []string{"volume", "vol"},
	Short:   "Manage volumes.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}
		return validateVolumeSelectors()
	},
}

func init() {
	volumesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter output by nodes optionally in ellipses pattern.")
	volumesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter output by drives optionally in ellipses pattern.")
	volumesCmd.PersistentFlags().BoolVarP(&allFlag, "all", "A", allFlag, "List all volumes.")
	volumesCmd.PersistentFlags().StringSliceVarP(&podNameArgs, "pod-name", "", podNameArgs, "Filter output by pod names optionally in ellipses pattern.")
	volumesCmd.PersistentFlags().StringSliceVarP(&podNSArgs, "pod-namespace", "", podNSArgs, "Filter output by pod namespaces optionally in ellipses pattern.")

	volumesCmd.AddCommand(volumesListCmd)
	volumesCmd.AddCommand(volumesPurgeCmd)
}
