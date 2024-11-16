// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"fmt"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// RefreshNodes refreshes the nodes provided in the input
func (client *Client) RefreshNodes(ctx context.Context, selectedNodes []string) (<-chan directpvtypes.NodeID, <-chan error, error) {
	if err := client.SyncNodes(ctx); err != nil {
		return nil, nil, err
	}

	nodes, err := client.NewNodeLister().
		NodeSelector(directpvtypes.ToLabelValues(selectedNodes)).
		Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(selectedNodes) != 0 && len(nodes) == 0 {
		suffix := ""
		if len(selectedNodes) > 1 {
			suffix = "s"
		}
		return nil, nil, fmt.Errorf("node%v %v not found", suffix, strings.Join(selectedNodes, ","))
	}

	nodeCh := make(chan directpvtypes.NodeID)
	errCh := make(chan error)

	go func() {
		defer close(nodeCh)
		defer close(errCh)

		nodeClient := client.Node()
		for i := range nodes {
			updateFunc := func() error {
				node, err := nodeClient.Get(ctx, nodes[i].Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				select {
				case nodeCh <- directpvtypes.NodeID(node.Name):
				case <-ctx.Done():
					return ctx.Err()
				}
				node.Spec.Refresh = true
				if _, err := nodeClient.Update(ctx, node, metav1.UpdateOptions{TypeMeta: types.NewNodeTypeMeta()}); err != nil {
					return err
				}
				return nil
			}
			if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
				errCh <- err
				return
			}
		}
	}()

	return nodeCh, errCh, nil
}
