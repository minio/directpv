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

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get drives and volumes.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}

		return validateGetCmd()
	},
}

func init() {
	addNodeFlag(getCmd, "Filter output by nodes")
	addDriveNameFlag(getCmd, "Filter output by drive names")
	addAllFlag(getCmd, "If present, list all objects")
	getCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", outputFormat, "Output format. One of: json|yaml|wide")
	getCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", noHeaders, "When using the default or custom-column output format, don't print headers (default print headers)")

	getCmd.AddCommand(getDrivesCmd)
	getCmd.AddCommand(getVolumesCmd)
}

func validateGetCmd() error {
	switch outputFormat {
	case "":
	case "wide":
		wideOutput = true
	case "yaml":
		yamlOutput = true
	case "json":
		jsonOutput = true
	default:
		return errors.New("--output flag value must be one of wide|json|yaml or empty")
	}

	printer = printYAML
	if jsonOutput {
		printer = printJSON
	}

	if err := validateNodeArgs(); err != nil {
		return err
	}

	if err := validateDriveNameArgs(); err != nil {
		return err
	}

	return nil
}
