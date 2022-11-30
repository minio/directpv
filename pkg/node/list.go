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
	"errors"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

var errUnsupportedSelector = errors.New("unsupported selector")

// ListNodeResult denotes list of node result.
type ListNodeResult struct {
	Node types.Node
	Err  error
}

// Lister is node lister.
type Lister struct {
	nodes          []directpvtypes.LabelValue
	nodeNames      []string
	maxObjects     int64
	ignoreNotFound bool
}

// NewLister creates new volume lister.
func NewLister() *Lister {
	return &Lister{
		maxObjects: k8s.MaxThreadCount,
	}
}

// NodeSelector adds filter listing by nodes.
func (lister *Lister) NodeSelector(nodes []directpvtypes.LabelValue) *Lister {
	lister.nodes = nodes
	return lister
}

// NodeNameSelector adds filter listing by node names.
func (lister *Lister) NodeNameSelector(nodeNames []string) *Lister {
	lister.nodeNames = nodeNames
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *Lister) MaxObjects(n int64) *Lister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore node not found error.
func (lister *Lister) IgnoreNotFound(b bool) *Lister {
	lister.ignoreNotFound = b
	return lister
}

// List returns channel to loop through node items.
func (lister *Lister) List(ctx context.Context) <-chan ListNodeResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.nodeNames) != 0

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey: lister.nodes,
	}
	labelSelector := directpvtypes.ToLabelSelector(labelMap)

	resultCh := make(chan ListNodeResult)
	go func() {
		defer close(resultCh)

		send := func(result ListNodeResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		if !getOnly {
			options := metav1.ListOptions{
				Limit:         lister.maxObjects,
				LabelSelector: labelSelector,
			}

			for {
				result, err := client.NodeClient().List(ctx, options)
				if err != nil {
					send(ListNodeResult{Err: err})
					return
				}

				for _, item := range result.Items {
					var found bool
					var values []string
					for i := range lister.nodeNames {
						if lister.nodeNames[i] == item.Name {
							found = true
						} else {
							values = append(values, lister.nodeNames[i])
						}
					}
					lister.nodeNames = values

					if len(lister.nodeNames) == 0 || found {
						if !send(ListNodeResult{Node: item}) {
							return
						}
					}
				}

				if result.Continue == "" {
					break
				}

				options.Continue = result.Continue
			}
		}

		for _, nodeName := range lister.nodeNames {
			node, err := client.NodeClient().Get(ctx, nodeName, metav1.GetOptions{})
			if err != nil {
				send(ListNodeResult{Err: err})
				return
			}
			if !send(ListNodeResult{Node: *node}) {
				return
			}
		}
	}()

	return resultCh
}

// Get returns list of nodes.
func (lister *Lister) Get(ctx context.Context) ([]types.Node, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	nodeList := []types.Node{}
	for result := range lister.List(ctx) {
		if result.Err != nil {
			return nodeList, result.Err
		}
		nodeList = append(nodeList, result.Node)
	}

	return nodeList, nil
}

// WatchEvent represents the node events.
type WatchEvent struct {
	Type watch.EventType
	Node *types.Node
}

// Watch looks for changes in NodeList and reports them.
func (lister *Lister) Watch(ctx context.Context) (<-chan WatchEvent, func(), error) {
	if len(lister.nodeNames) > 0 {
		return nil, nil, errUnsupportedSelector
	}

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey: lister.nodes,
	}
	nodeWatchInterface, err := client.NodeClient().Watch(ctx, metav1.ListOptions{
		LabelSelector: directpvtypes.ToLabelSelector(labelMap),
	})
	if err != nil {
		return nil, nil, err
	}
	stopFn := nodeWatchInterface.Stop

	watchCh := make(chan WatchEvent)
	go func() {
		defer close(watchCh)
		resultCh := nodeWatchInterface.ResultChan()
		for {
			result, ok := <-resultCh
			if !ok {
				break
			}
			unstructured := result.Object.(*unstructured.Unstructured)
			var node types.Node
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &node)
			if err != nil {
				klog.ErrorS(err, "unable to convert unstructured object %s", unstructured.GetName())
				break
			}
			watchCh <- WatchEvent{
				Type: result.Type,
				Node: &node,
			}
		}
	}()

	return watchCh, stopFn, nil
}
