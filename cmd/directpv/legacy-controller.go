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
	"github.com/minio/directpv/pkg/csi/controller"
	pkgidentity "github.com/minio/directpv/pkg/identity"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var legacyControllerCmd = &cobra.Command{
	Use:           consts.LegacyControllerServerName,
	Short:         "Start legacy controller server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startLegacyController(c.Context())
	},
}

func startLegacyController(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	idServer, err := pkgidentity.NewServer(legacyclient.Identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("Legacy identity server started")

	ctrlServer := controller.NewLegacyServer()
	klog.V(3).Infof("Legacy controller server started")

	errCh := make(chan error)

	go func() {
		if err := runServers(ctx, csiEndpoint, idServer, ctrlServer, nil); err != nil {
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
