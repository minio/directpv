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
	"github.com/minio/directpv/pkg/csi/node"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/drive"
	pkgidentity "github.com/minio/directpv/pkg/identity"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var metricsPort = consts.MetricsPort

var nodeServerCmd = &cobra.Command{
	Use:           consts.NodeServerName,
	Short:         "Start node server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		if err := device.Sync(c.Context(), nodeID); err != nil {
			return err
		}
		return startNodeServer(c.Context())
	},
}

func init() {
	nodeServerCmd.PersistentFlags().IntVar(&metricsPort, "metrics-port", metricsPort, "Metrics port at "+consts.AppPrettyName+" exports metrics data")
}

func startNodeServer(ctx context.Context) error {
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
		volume.StartController(ctx, nodeID)
		errCh <- errors.New("volume controller stopped")
	}()

	go func() {
		drive.StartController(ctx, nodeID)
		errCh <- errors.New("drive controller stopped")
	}()

	nodeServer := node.NewServer(
		ctx,
		identity,
		nodeID,
		rack,
		zone,
		region,
		metricsPort,
	)
	klog.V(3).Infof("Node server started")

	if err = sys.Mkdir(consts.MountRootDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
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
