// This file is part of MinIO Kubernetes Cloud
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
	"fmt"
	"strings"

	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	"github.com/minio/jbod-csi-driver/pkg/controller"
	id "github.com/minio/jbod-csi-driver/pkg/identity"
	"github.com/minio/jbod-csi-driver/pkg/node"
	"github.com/minio/jbod-csi-driver/pkg/volume"

	"github.com/golang/glog"
	"github.com/minio/minio/pkg/ellipses"
)

func driver(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no base paths provided for jbods")
	}

	if mode != "controller" && mode != "node" && mode != "all" {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("identity server started")

	basePaths := []string{}
	for _, a := range args {
		if ellipses.HasEllipses(a) {
			p, _ := ellipses.FindEllipsesPatterns(a)
			patterns := p.Expand()
			for _, outer := range patterns {
				basePaths = append(basePaths, strings.Join(outer, ""))
			}
		} else {
			basePaths = append(basePaths, a)
		}
	}

	volume.InitializeFactory(basePaths)
	volume.InitializeClient(identity)

	node, err := node.NewNodeServer(identity, nodeID, rack, zone, region)
	if err != nil {
		return err
	}

	if mode == "node" || mode == "all" {
		glog.V(5).Infof("node server started")
	} else {
		node = nil
	}

	ctrlServer, err := controller.NewControllerServer(identity, nodeID, rack, zone, region)
	if err != nil {
		return err
	}

	if mode == "controller" || mode == "all" {
		glog.V(5).Infof("controller manager started")
	} else {
		ctrlServer = nil
	}

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(endpoint, idServer, ctrlServer, node)
	s.Wait()

	return nil
}
