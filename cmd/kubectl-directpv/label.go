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

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/spf13/cobra"
)

var labels []admin.Label

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Set labels to drives and volumes",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			parent.PersistentPreRunE(parent, args)
		}
		return validateLabelCmd()
	},
}

func init() {
	setFlagOpts(labelCmd)

	addNodesFlag(labelCmd, "If present, filter objects from given nodes")
	addDrivesFlag(labelCmd, "If present, filter objects by given drive names")
	addAllFlag(labelCmd, "If present, select all objects")
	addDryRunFlag(labelCmd, "Run in dry run mode")

	labelCmd.AddCommand(labelDrivesCmd)
	labelCmd.AddCommand(labelVolumesCmd)
}

func validateLabelCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}

	return validateDriveNameArgs()
}

func validateLabelCmdArgs(args []string) (labels []admin.Label, err error) {
	if len(args) == 0 {
		return nil, errors.New("at least one label must be provided")
	}

	for _, arg := range args {
		var label admin.Label
		tokens := strings.Split(arg, "=")
		switch len(tokens) {
		case 1:
			if !strings.HasSuffix(arg, "-") {
				return nil, fmt.Errorf("argument %v must end with '-' to remove label", arg)
			}
			label.Remove = true
			if label.Key, err = types.NewLabelKey(arg[:len(arg)-1]); err != nil {
				return nil, err
			}
		case 2:
			if label.Key, err = types.NewLabelKey(tokens[0]); err != nil {
				return nil, err
			}
			if label.Value, err = types.NewLabelValue(tokens[1]); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("argument must be formatted k=v or k-")
		}
		labels = append(labels, label)
	}

	return labels, nil
}
