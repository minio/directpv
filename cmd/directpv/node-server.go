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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/drive"
	pkgidentity "github.com/minio/directpv/pkg/identity"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var metricsPort = consts.MetricsPort

var nodeServerCmd = &cobra.Command{
	Use:           "node-server",
	Short:         "Start node server of " + consts.AppPrettyName + ".",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startNodeServer(c.Context(), args)
	},
}

func init() {
	nodeServerCmd.PersistentFlags().IntVarP(&metricsPort, "metrics-port", "", metricsPort, "Metrics port at "+consts.AppPrettyName+" exports metrics data")
}

func startNodeServer(ctx context.Context, args []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	idServer, err := pkgidentity.NewServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("Identity server started")

	errCh := make(chan error)

	go func() {
		if err := volume.StartController(ctx, nodeID); err != nil {
			klog.ErrorS(err, "unable to start volume controller")
			errCh <- err
		}
	}()

	go func() {
		if err := drive.StartController(ctx, nodeID); err != nil {
			klog.ErrorS(err, "unable to start drive controller")
			errCh <- err
		}
	}()

	var nodeServer csi.NodeServer
	nodeServer, err = node.NewServer(
		ctx,
		identity,
		nodeID,
		rack,
		zone,
		region,
		metricsPort,
	)
	if err != nil {
		return err
	}
	klog.V(3).Infof("Node server started")

	if err = os.Mkdir(consts.MountRootDir, 0o777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	go func() {
		if err := runServers(ctx, csiEndpoint, idServer, nil, nodeServer); err != nil {
			klog.ErrorS(err, "unable to start GRPC servers")
			errCh <- err
		}
	}()

	go func() {
		if err := serveReadinessEndpoint(ctx); err != nil {
			klog.ErrorS(err, "unable to start readiness endpoint")
			errCh <- err
		}
	}()

	return <-errCh
}
