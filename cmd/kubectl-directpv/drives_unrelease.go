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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var unreleaseDrivesCmd = &cobra.Command{
	Use:   "unrelease",
	Short: utils.BinaryNameTransform("unrelease drives in the {{ . }} cluster"),
	Long:  "",
	RunE: func(c *cobra.Command, args []string) error {
		return nil
	},
	Deprecated:            utils.BinaryNameTransform("please use `kubectl {{ . }} drives release` which will umount and make the drives `Available`"),
	Aliases:               []string{},
	Hidden:                true,
	DisableFlagsInUseLine: true,
}
