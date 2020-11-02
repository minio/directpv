// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package cmd

import (
	"context"

	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/minio/direct-csi/pkg/controller"
	id "github.com/minio/direct-csi/pkg/identity"
	"github.com/minio/direct-csi/pkg/node"

	"github.com/golang/glog"
)

func driver(_ []string) error {
	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("identity server started")

	drives, err := node.FindDrives(context.Background(), nodeID)
	if err != nil {
		return err
	}

	drivePaths := []string{}
	for _, drive := range drives {
		drivePaths = append(drivePaths, drive.Path)
	}
	glog.V(5).Info(drivePaths)

	basePaths := node.MountDevices(drivePaths)
	node, err := node.NewNodeServer(identity, nodeID, rack, zone, region, basePaths)
	if err != nil {
		return err
	}
	glog.V(5).Infof("node server started")

	ctrlServer, err := controller.NewControllerServer(identity, nodeID, rack, zone, region)
	if err != nil {
		return err
	}
	glog.V(5).Infof("controller manager started")

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(endpoint, idServer, ctrlServer, node)
	s.Wait()

	return nil
}
