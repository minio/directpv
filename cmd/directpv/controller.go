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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/csi/controller"
	pkgidentity "github.com/minio/directpv/pkg/csi/identity"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var controllerCmd = &cobra.Command{
	Use:           consts.ControllerServerName,
	Short:         "Start controller server.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, _ []string) error {
		klog.InfoS("Starting DirectPV controller server", "version", Version)
		return startController(c.Context())
	},
}

func startController(ctx context.Context) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	capabilities := pkgidentity.GetDefaultPluginCapabilities()
	capabilities = append(capabilities, &csi.PluginCapability{
		Type: &csi.PluginCapability_VolumeExpansion_{
			VolumeExpansion: &csi.PluginCapability_VolumeExpansion{
				Type: csi.PluginCapability_VolumeExpansion_ONLINE,
			},
		},
	})
	idServer, err := pkgidentity.NewServer(identity, Version, capabilities)
	if err != nil {
		return err
	}
	klog.V(3).Infof("Identity server started")

	ctrlServer := controller.NewServer()
	klog.V(3).Infof("Controller server started")

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
