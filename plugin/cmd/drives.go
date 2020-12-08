// +build plugin

/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
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

package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

const (
	csiDriveDesc = `
 drives command allows managing Drives on MinIO DirectCSI.`
)

func newDrivesCmd(out io.Writer, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drives",
		Short: "Mangage Drives on DirectCSI",
		Long:  csiDriveDesc,
	}
	cmd.AddCommand(newDrivesAddCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newDrivesRemoveCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newDrivesListCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	return cmd
}
