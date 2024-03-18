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

	"github.com/minio/directpv/pkg/consts"
	pkgidentity "github.com/minio/directpv/pkg/csi/identity"
	"github.com/minio/directpv/pkg/csi/node"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var legacyNodeServerCmd = &cobra.Command{
	Use:           consts.LegacyNodeServerName,
	Short:         "Start legacy node server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, _ []string) error {
		return startLegacyNodeServer(c.Context())
	},
}

func startLegacyNodeServer(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	idServer, err := pkgidentity.NewServer(legacyclient.Identity, Version, pkgidentity.GetDefaultPluginCapabilities())
	if err != nil {
		return err
	}
	klog.V(3).Infof("Legacy identity server started")

	errCh := make(chan error)

	nodeServer := node.NewLegacyServer(nodeID, rack, zone, region)
	klog.V(3).Infof("Legacy node server started")

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
