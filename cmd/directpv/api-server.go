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

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var apiServer = &cobra.Command{
	Use:           "api-server",
	Short:         "Start API server of " + consts.AppPrettyName + ".",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startAPIServer(c.Context(), args)
	},
	// FIXME: Add help messages
}

func init() {
	apiServer.PersistentFlags().IntVarP(&apiPort, "port", "", apiPort, "port for "+consts.AppPrettyName+" API server")
}

func startAPIServer(ctx context.Context, args []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)
	go func() {
		if err := admin.ServeAPIServer(ctx, apiPort); err != nil {
			klog.ErrorS(err, "unable to run API server")
			errCh <- err
		}
	}()

	return <-errCh
}
