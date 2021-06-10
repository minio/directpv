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

var drivesAccessTierCmd = &cobra.Command{
	Use:   "access-tier",
	Short: "tag/untag DirectCSI drives based on thier access-tiers",
	Long:  "",
	Aliases: []string{
		"accesstier",
	},
}

func init() {
	drivesAccessTierCmd.AddCommand(accessTierSet)
	drivesAccessTierCmd.AddCommand(accessTierUnset)
}
