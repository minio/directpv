// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"fmt"

	ctrl "github.com/minio/direct-csi/pkg/controller"
	"github.com/minio/direct-csi/pkg/converter"
	id "github.com/minio/direct-csi/pkg/identity"
	"github.com/minio/direct-csi/pkg/node"
	"github.com/minio/direct-csi/pkg/node/discovery"
	"github.com/minio/direct-csi/pkg/utils/grpc"
	"github.com/minio/direct-csi/pkg/volume"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

func run(ctx context.Context, args []string) error {

	// Start conversion webserver
	if err := converter.ServeConversionWebhook(ctx); err != nil {
		klog.V(3).Infof("Stopped serving conversion webhook: %v", err)
	}

	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("identity server started")

	var nodeSrv csi.NodeServer
	if driver {
		discovery, err := discovery.NewDiscovery(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		klog.Infof("running drive discovery")
		if err := discovery.Init(ctx, loopBackOnly); err != nil {
			return fmt.Errorf("Error while initializing drive discovery: %v", err)
		}
		klog.V(5).Infof("Drive discovery finished")

		// Check if the volume objects are migrated and CRDs versions are in-sync
		volume.SyncVolumes(ctx, nodeID)
		klog.V(5).Infof("Volumes sync completed")

		nodeSrv, err = node.NewNodeServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		klog.V(5).Infof("node server started")
	}

	var ctrlServer csi.ControllerServer
	if controller {
		ctrlServer, err = ctrl.NewControllerServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		klog.V(5).Infof("controller manager started")
	}

	return grpc.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv)
}
