/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package main

import (
	"github.com/spf13/cobra"
)

var unreleaseDrivesCmd = &cobra.Command{
	Use:   "unrelease",
	Short: binaryNameTransform("unrelease drives in the {{ . }} cluster"),
	Long:  "",
	RunE: func(c *cobra.Command, args []string) error {
		return nil
	},
	Deprecated:            binaryNameTransform("please use `kubectl {{ . }} drives release` which will umount and make the drives `Available`"),
	Aliases:               []string{},
	Hidden:                true,
	DisableFlagsInUseLine: true,
}
