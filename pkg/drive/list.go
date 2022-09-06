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

package drive

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// ListDriveResult denotes list of drive result.
type ListDriveResult struct {
	Drive types.Drive
	Err   error
}

// ListDrives lists drives.
func ListDrives(ctx context.Context, nodes, drives, accessTiers []types.LabelValue, maxObjects int64) (<-chan ListDriveResult, error) {
	labelMap := map[types.LabelKey][]types.LabelValue{
		types.PathLabelKey:       drives,
		types.NodeLabelKey:       nodes,
		types.AccessTierLabelKey: accessTiers,
	}
	labelSelector := types.ToLabelSelector(labelMap)

	resultCh := make(chan ListDriveResult)
	go func() {
		defer close(resultCh)
		klog.V(5).InfoS("Listing drives", "limit", maxObjects, "selectors", labelSelector)

		send := func(result ListDriveResult) bool {
			select {
			case <-ctx.Done():
				return false
			case resultCh <- result:
				return true
			}
		}

		options := metav1.ListOptions{
			Limit:         maxObjects,
			LabelSelector: labelSelector,
		}
		for {
			result, err := client.DriveClient().List(ctx, options)
			if err != nil {
				send(ListDriveResult{Err: err})
				return
			}

			for _, item := range result.Items {
				if !send(ListDriveResult{Drive: item}) {
					return
				}
			}

			if result.Continue == "" {
				return
			}

			options.Continue = result.Continue
		}
	}()

	return resultCh, nil
}

// GetDriveList gets list of drives.
func GetDriveList(ctx context.Context, nodes, drives, accessTiers []types.LabelValue) ([]types.Drive, error) {
	resultCh, err := ListDrives(ctx, nodes, drives, accessTiers, k8s.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	driveList := []types.Drive{}
	for result := range resultCh {
		if result.Err != nil {
			return driveList, result.Err
		}
		driveList = append(driveList, result.Drive)
	}

	return driveList, nil
}

// ProcessDrives processes the drives by applying the provided filter functions
func ProcessDrives(
	ctx context.Context,
	resultCh <-chan ListDriveResult,
	matchFunc func(*types.Drive) bool,
	applyFunc func(*types.Drive) error,
	processFunc func(context.Context, *types.Drive) error,
	writer io.Writer,
	dryRun bool,
) error {
	objectCh := make(chan k8s.ObjectResult)
	go func() {
		defer close(objectCh)
		for result := range resultCh {
			var oresult k8s.ObjectResult
			if result.Err != nil {
				oresult.Err = result.Err
			} else {
				drive := result.Drive
				oresult.Object = &drive
			}

			select {
			case <-ctx.Done():
				return
			case objectCh <- oresult:
			}
		}
	}()

	return k8s.ProcessObjects(
		ctx,
		objectCh,
		func(object runtime.Object) bool {
			return matchFunc(object.(*types.Drive))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*types.Drive))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*types.Drive))
		},
		writer,
		dryRun,
	)
}

// DrivesListerWatcher is the lister watcher for drives.
func DrivesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", types.NodeLabelKey, types.NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.DriveResource,
		"",
		optionsModifier,
	)
}
