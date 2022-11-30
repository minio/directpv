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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/node"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var nodeControllerCmd = &cobra.Command{
	Use:           consts.NodeControllerName,
	Short:         "Start node controller.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		if err := node.Sync(c.Context(), nodeID, true); err != nil {
			return err
		}
		return startNodeController(c.Context(), args)
	},
}

func startNodeController(ctx context.Context, args []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	go func() {
		if err := node.StartController(ctx, nodeID); err != nil {
			klog.ErrorS(err, "unable to start node controller")
			errCh <- err
		}
	}()

	go func() {
		if err := initrequest.StartController(ctx, nodeID); err != nil {
			klog.ErrorS(err, "unable to start initrequest controller")
			errCh <- err
		}
	}()

	return <-errCh
}
