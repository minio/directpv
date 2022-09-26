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
	"github.com/minio/directpv/pkg/rest"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var nodeAPIServer = &cobra.Command{
	Use:           "node-api-server",
	Short:         "Start Node API server of " + consts.AppPrettyName + ".",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startNodeAPIServer(c.Context(), args)
	},
	// FIXME: Add help messages
}

func init() {
	nodeAPIServer.PersistentFlags().IntVarP(&nodeAPIPort, "port", "", nodeAPIPort, "port for "+consts.AppPrettyName+" Node API server")
}

// ServeNodeAPIServer(ctx context.Context, nodeAPIPort int, identity, nodeID, rack, zone, region string) error {
func startNodeAPIServer(ctx context.Context, args []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	if err := os.Mkdir(consts.MountRootDir, 0o777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	errCh := make(chan error)
	go func() {
		if err := rest.ServeNodeAPIServer(ctx,
			nodeAPIPort,
			identity,
			kubeNodeName,
			rack,
			zone,
			region,
		); err != nil {
			klog.ErrorS(err, "unable to run node API server")
			errCh <- err
		}
	}()

	return <-errCh
}
