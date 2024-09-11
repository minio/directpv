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
	"sort"
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List drives and volumes",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}
		return validateListCmd()
	},
}

func init() {
	setFlagOpts(listCmd)

	addNodesFlag(listCmd, "Filter output by nodes")
	addDrivesFlag(listCmd, "Filter output by drive names")
	addOutputFormatFlag(listCmd, "Output format. One of: json|yaml|wide")
	listCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", noHeaders, "When using the default or custom-column output format, don't print headers (default print headers)")

	listCmd.AddCommand(listDrivesCmd)
	listCmd.AddCommand(listVolumesCmd)
}

func validateListCmd() error {
	if err := validateOutputFormat(true); err != nil {
		return err
	}
	if err := validateNodeArgs(); err != nil {
		return err
	}
	if err := validateDriveNameArgs(); err != nil {
		return err
	}
	return validateLabelArgs()
}

func labelsToString(labels map[string]string) string {
	var labelsArray []string
	for k, v := range labels {
		if !types.LabelKey(k).IsReserved() {
			k = strings.TrimPrefix(k, consts.GroupName+"/")
			labelsArray = append(labelsArray, fmt.Sprintf("%s=%v", k, v))
		}
	}
	if len(labelsArray) == 0 {
		return "-"
	}
	sort.Strings(labelsArray)
	return strings.Join(labelsArray, ",")
}
