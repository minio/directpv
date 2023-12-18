// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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
	"github.com/spf13/cobra"
)

var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: fmt.Sprintf("purge %v resources", consts.AppPrettyName),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}
		return validatePurgeCmd()
	},
}

func init() {
	setFlagOpts(purgeCmd)

	addNodesFlag(purgeCmd, "If present, filter objects from given nodes")
	addAllFlag(purgeCmd, "If present, select all objects")
	addDryRunFlag(purgeCmd, "Run in dry run mode")

	purgeCmd.AddCommand(purgeJobsCmd)
}

func validatePurgeCmd() error {
	return validateNodeArgs()
}
