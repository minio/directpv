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
	"strings"

	ctrl "github.com/minio/direct-csi/pkg/controller"
	id "github.com/minio/direct-csi/pkg/identity"
	"github.com/minio/direct-csi/pkg/node"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/minio/minio/pkg/ellipses"
)

func run(ctx context.Context, args []string) error {
	utils.Init()

	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("identity server started")

	basePaths := []string{}
	for _, a := range args {
		if ellipses.HasEllipses(a) {
			p, err := ellipses.FindEllipsesPatterns(a)
			if err != nil {
				return err
			}
			patterns := p.Expand()
			for _, outer := range patterns {
				basePaths = append(basePaths, strings.Join(outer, ""))
			}
		} else {
			basePaths = append(basePaths, a)
		}
	}

	var nodeSrv csi.NodeServer
	if driver {
		nodeSrv, err = node.NewNodeServer(ctx, identity, nodeID, rack, zone, region, basePaths, procfs)
		if err != nil {
			return err
		}
		glog.V(5).Infof("node server started")
	}

	var ctrlServer csi.ControllerServer
	if controller {
		ctrlServer, err = ctrl.NewControllerServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		glog.V(5).Infof("controller manager started")
	}

	return utils.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv)
}
