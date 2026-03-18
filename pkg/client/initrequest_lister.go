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

package client

import (
	"context"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// ListInitRequestResult denotes list of initrequest result.
type ListInitRequestResult struct {
	InitRequest types.InitRequest
	Err         error
}

// InitRequestLister is initRequest lister.
type InitRequestLister struct {
	nodes             []directpvtypes.LabelValue
	requestIDs        []directpvtypes.LabelValue
	initRequestNames  []string
	maxObjects        int64
	ignoreNotFound    bool
	initRequestClient types.LatestInitRequestInterface
}

// NewInitRequestLister creates new volume lister.
func (c Client) NewInitRequestLister() *InitRequestLister {
	return &InitRequestLister{
		maxObjects:        k8s.MaxThreadCount,
		initRequestClient: c.InitRequest(),
	}
}

// NodeSelector adds filter listing by nodes.
func (lister *InitRequestLister) NodeSelector(nodes []directpvtypes.LabelValue) *InitRequestLister {
	lister.nodes = nodes
	return lister
}

// RequestIDSelector adds filter listing by its request IDs.
func (lister *InitRequestLister) RequestIDSelector(requestIDs []directpvtypes.LabelValue) *InitRequestLister {
	lister.requestIDs = requestIDs
	return lister
}

// InitRequestNameSelector adds filter listing by InitRequestNames.
func (lister *InitRequestLister) InitRequestNameSelector(initRequestNames []string) *InitRequestLister {
	lister.initRequestNames = initRequestNames
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *InitRequestLister) MaxObjects(n int64) *InitRequestLister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore not found error.
func (lister *InitRequestLister) IgnoreNotFound(b bool) *InitRequestLister {
	lister.ignoreNotFound = b
	return lister
}

// List returns channel to loop through initrequest items.
func (lister *InitRequestLister) List(ctx context.Context) <-chan ListInitRequestResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.requestIDs) == 0 &&
		len(lister.initRequestNames) != 0

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey:      lister.nodes,
		directpvtypes.RequestIDLabelKey: lister.requestIDs,
	}
	labelSelector := directpvtypes.ToLabelSelector(labelMap)

	resultCh := make(chan ListInitRequestResult)
	go func() {
		defer close(resultCh)

		send := func(result ListInitRequestResult) bool {
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
				result, err := lister.initRequestClient.List(ctx, options)
				if err != nil {
					send(ListInitRequestResult{Err: err})
					return
				}

				for _, item := range result.Items {
					var found bool
					var values []string
					for i := range lister.initRequestNames {
						if lister.initRequestNames[i] == item.Name {
							found = true
						} else {
							values = append(values, lister.initRequestNames[i])
						}
					}
					lister.initRequestNames = values

					if len(lister.initRequestNames) == 0 || found {
						if !send(ListInitRequestResult{InitRequest: item}) {
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

		for _, initRequestName := range lister.initRequestNames {
			initRequest, err := lister.initRequestClient.Get(ctx, initRequestName, metav1.GetOptions{})
			if err != nil {
				send(ListInitRequestResult{Err: err})
				return
			}
			if !send(ListInitRequestResult{InitRequest: *initRequest}) {
				return
			}
		}
	}()

	return resultCh
}

// Get returns list of initrequest.
func (lister *InitRequestLister) Get(ctx context.Context) ([]types.InitRequest, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	initRequestList := []types.InitRequest{}
	for result := range lister.List(ctx) {
		if result.Err != nil {
			return initRequestList, result.Err
		}
		initRequestList = append(initRequestList, result.InitRequest)
	}

	return initRequestList, nil
}

// Watch looks for changes in InitRequestList and reports them.
func (lister *InitRequestLister) Watch(ctx context.Context) (<-chan WatchEvent[*types.InitRequest], func(), error) {
	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey:      lister.nodes,
		directpvtypes.RequestIDLabelKey: lister.requestIDs,
	}
	initRequestWatchInterface, err := lister.initRequestClient.Watch(ctx, metav1.ListOptions{
		LabelSelector: directpvtypes.ToLabelSelector(labelMap),
	})
	if err != nil {
		return nil, nil, err
	}
	stopFn := initRequestWatchInterface.Stop

	watchCh := make(chan WatchEvent[*types.InitRequest])
	go func() {
		defer close(watchCh)
		resultCh := initRequestWatchInterface.ResultChan()
		for {
			result, ok := <-resultCh
			if !ok {
				break
			}
			unstructured := result.Object.(*unstructured.Unstructured)
			var initRequest types.InitRequest
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, &initRequest)
			if err != nil {
				watchCh <- WatchEvent[*types.InitRequest]{
					Type: result.Type,
					Err:  fmt.Errorf("unable to convert unstructured object %s; %w", unstructured.GetName(), err),
				}
				continue
			}
			if len(lister.initRequestNames) > 0 && !utils.Contains(lister.initRequestNames, initRequest.Name) {
				continue
			}
			watchCh <- WatchEvent[*types.InitRequest]{
				Type: result.Type,
				Item: &initRequest,
			}
		}
	}()

	return watchCh, stopFn, nil
}
