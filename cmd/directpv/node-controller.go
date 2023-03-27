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
	"context"
	"errors"
	"os"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/sys"
	"github.com/spf13/cobra"
)

var nodeControllerCmd = &cobra.Command{
	Use:           consts.NodeControllerName,
	Short:         "Start node controller.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		if err := sys.Mkdir(consts.MountRootDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := node.Sync(c.Context(), nodeID); err != nil {
			return err
		}
		return startNodeController(c.Context())
	},
}

func startNodeController(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	go func() {
		node.StartController(ctx, nodeID)
		errCh <- errors.New("node controller stopped")
	}()

	go func() {
		initrequest.StartController(
			ctx,
			nodeID,
			identity,
			rack,
			zone,
			region,
		)
		errCh <- errors.New("initrequest controller stopped")
	}()

	return <-errCh
}
