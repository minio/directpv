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

package node

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
)

// LegacyServer denotes legacy node server.
type LegacyServer struct {
	Server
}

// NewLegacyServer creates legacy node server.
func NewLegacyServer(nodeID directpvtypes.NodeID, rack, zone, region string) *LegacyServer {
	return &LegacyServer{Server: newServer("direct-csi-min-io", nodeID, rack, zone, region)}
}

// NodeGetInfo gets node information.
// reference: https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetinfo
func (server *LegacyServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	topology := &csi.Topology{
		Segments: map[string]string{
			"direct.csi.min.io/identity": server.identity,
			"direct.csi.min.io/rack":     server.rack,
			"direct.csi.min.io/region":   server.region,
			"direct.csi.min.io/zone":     server.zone,
			"direct.csi.min.io/node":     string(server.nodeID),
		},
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             string(server.nodeID),
		MaxVolumesPerNode:  100,
		AccessibleTopology: topology,
	}, nil
}
