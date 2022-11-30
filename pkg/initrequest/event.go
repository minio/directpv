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

package initrequest

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

type initRequestEventHandler struct {
	nodeID directpvtypes.NodeID
}

func newInitRequestEventHandler(nodeID directpvtypes.NodeID) *initRequestEventHandler {
	return &initRequestEventHandler{
		nodeID: nodeID,
	}
}

func (handler *initRequestEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := fmt.Sprintf("%s=%s", directpvtypes.NodeLabelKey, handler.nodeID)
	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.InitRequestResource,
		"",
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		},
	)
}

func (handler *initRequestEventHandler) Name() string {
	return "initrequest"
}

func (handler *initRequestEventHandler) ObjectType() runtime.Object {
	return &types.InitRequest{}
}

func (handler *initRequestEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	return nil
}

// StartController starts node controller.
func StartController(ctx context.Context, nodeID directpvtypes.NodeID) error {
	listener := listener.NewListener(newInitRequestEventHandler(nodeID), "initrequest-controller", string(nodeID), 40)
	return listener.Run(ctx)
}
