// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package client

import (
	"context"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncNodes compares the csinodes with directpvnode list and syncs them
// It adds the missing nodes and deletes the non-existing nodes
func SyncNodes(ctx context.Context) (err error) {
	csiNodes, err := k8s.GetCSINodes(ctx)
	if err != nil {
		return fmt.Errorf("unable to get CSI nodes; %w", err)
	}

	nodes, err := NewNodeLister().Get(ctx)
	if err != nil {
		return fmt.Errorf("unable to get nodes; %w", err)
	}

	var nodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}

	// Add missing nodes.
	for _, csiNode := range csiNodes {
		if !utils.Contains(nodeNames, csiNode) {
			node := types.NewNode(directpvtypes.NodeID(csiNode), nil)
			node.Spec.Refresh = true
			if _, err = NodeClient().Create(ctx, node, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("unable to create node %v; %w", csiNode, err)
			}
		}
	}

	// Remove non-existing nodes.
	for _, nodeName := range nodeNames {
		if !utils.Contains(csiNodes, nodeName) {
			if err = NodeClient().Delete(ctx, nodeName, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("unable to remove non-existing node %v; %w", nodeName, err)
			}
		}
	}

	return nil
}
