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
	"context"
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:           "move SRC-DRIVE DEST-DRIVE",
	Aliases:       []string{"mv"},
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Move volumes excluding data from source drive to destination drive on a same node",
	Example: strings.ReplaceAll(
		`1. Move volumes from drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 to drive 834e8f4c-14f4-49b9-9b77-e8ac854108d5
   $ kubectl {PLUGIN_NAME} drives move af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 834e8f4c-14f4-49b9-9b77-e8ac854108d5`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 2 {
			eprintf(true, "only one source and one destination drive must be provided\n")
			os.Exit(-1)
		}

		src := strings.TrimSpace(args[0])
		if src == "" {
			eprintf(true, "empty source drive\n")
			os.Exit(-1)
		}

		dest := strings.TrimSpace(args[1])
		if dest == "" {
			eprintf(true, "empty destination drive\n")
			os.Exit(-1)
		}

		moveMain(c.Context(), directpvtypes.DriveID(src), directpvtypes.DriveID(dest))
	},
}

func moveMain(ctx context.Context, src, dest directpvtypes.DriveID) {
	err := adminClient.Move(
		ctx,
		admin.MoveArgs{
			Source:      src,
			Destination: dest,
		},
		logFunc,
	)
	if err != nil {
		eprintf(true, "%v\n", err)
		os.Exit(1)
	}
}
