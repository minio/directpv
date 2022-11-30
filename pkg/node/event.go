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

package node

import (
	"context"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type nodeEventHandler struct {
	nodeID directpvtypes.NodeID
}

func newNodeEventHandler(nodeID directpvtypes.NodeID) *nodeEventHandler {
	return &nodeEventHandler{
		nodeID: nodeID,
	}
}

func (handler *nodeEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, handler.nodeID)
	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.NodeResource,
		"",
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		},
	)
}

func (handler *nodeEventHandler) Name() string {
	return "node"
}

func (handler *nodeEventHandler) ObjectType() runtime.Object {
	return &types.Node{}
}

func (handler *nodeEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.UpdateEvent, listener.SyncEvent:
		node := args.Object.(*types.Node)
		if node.Spec.Refresh {
			return Sync(ctx, directpvtypes.NodeID(node.Name), false)
		}
	case listener.AddEvent, listener.DeleteEvent:
	}
	return nil
}

// StartController starts node controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID) error {
	listener := listener.NewListener(newNodeEventHandler(nodeID), "node-controller", string(nodeID), 40)
	return listener.Run(ctx)
}
