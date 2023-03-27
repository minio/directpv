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
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

type label struct {
	key    types.LabelKey
	value  types.LabelValue
	remove bool
}

func (l label) String() string {
	if l.value == "" {
		return string(l.key)
	}
	return string(l.key) + ":" + string(l.value)
}

var labels []label

var errInvalidKVPair = errors.New("invalid kv pair")

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
	labelCmd.Flags().SortFlags = false
	labelCmd.InheritedFlags().SortFlags = false
	labelCmd.LocalFlags().SortFlags = false
	labelCmd.LocalNonPersistentFlags().SortFlags = false
	labelCmd.NonInheritedFlags().SortFlags = false
	labelCmd.PersistentFlags().SortFlags = false

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

func parseLabel(kv string) (labelKey types.LabelKey, labelValue types.LabelValue, err error) {
	kvParts := strings.Split(kv, "=")
	if len(kvParts) != 2 {
		err = errInvalidKVPair
		return
	}
	labelKey, err = types.NewLabelKey(kvParts[0])
	if err != nil {
		return
	}
	labelValue, err = types.NewLabelValue(kvParts[1])
	if err != nil {
		return
	}
	return
}

func validateLabelCmdArgs(args []string) (labels []label, err error) {
	parse := func(arg string) (label label, err error) {
		switch {
		case strings.Contains(arg, "="):
			label.key, label.value, err = parseLabel(arg)
		case strings.HasSuffix(arg, "-"):
			label.remove = true
			label.key, err = types.NewLabelKey(arg[:len(arg)-1])
		default:
			err = errInvalidKVPair
		}
		return
	}
	if len(args) == 0 {
		utils.Eprintf(quietFlag, false, "Please specify k=v|k- argument to set or unset label\n")
		return nil, errInvalidKVPair
	}
	for _, arg := range args {
		label, err := parse(arg)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return labels, nil
}
