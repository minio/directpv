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
	"os"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/sys"
	"github.com/spf13/cobra"
)

var nodeAPIPort = consts.NodeAPIPort

var nodeAPIServerCmd = &cobra.Command{
	Use:           consts.NodeAPIServerName,
	Short:         "Start node API server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		if err := sys.Mkdir(consts.MountRootDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		return admin.StartNodeAPIServer(
			c.Context(),
			nodeAPIPort,
			identity,
			nodeID,
			rack,
			zone,
			region,
		)
	},
}

func init() {
	nodeAPIServerCmd.PersistentFlags().IntVar(&nodeAPIPort, "port", nodeAPIPort, "Node API server port number")
}
