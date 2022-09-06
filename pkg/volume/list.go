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

package volume

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

// ListVolumeResult denotes list of volume result.
type ListVolumeResult struct {
	Volume types.Volume
	Err    error
}

// ListVolumes lists volumes.
func ListVolumes(ctx context.Context, nodes, drives, podNames, podNSs []types.LabelValue, maxObjects int64) (<-chan ListVolumeResult, error) {
	labelMap := map[types.LabelKey][]types.LabelValue{
		types.DrivePathLabelKey: drives,
		types.NodeLabelKey:      nodes,
		types.PodNameLabelKey:   podNames,
		types.PodNSLabelKey:     podNSs,
	}
	labelSelector := types.ToLabelSelector(labelMap)

	resultCh := make(chan ListVolumeResult)
	go func() {
		defer close(resultCh)
		klog.V(5).InfoS("Listing volumes", "limit", maxObjects, "selectors", labelSelector)

		send := func(result ListVolumeResult) bool {
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
			result, err := client.VolumeClient().List(ctx, options)
			if err != nil {
				send(ListVolumeResult{Err: err})
				return
			}

			for _, item := range result.Items {
				if !send(ListVolumeResult{Volume: item}) {
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

// GetVolumeList gets list of volumes.
func GetVolumeList(ctx context.Context, nodes, drives, podNames, podNSs []types.LabelValue) ([]types.Volume, error) {
	resultCh, err := ListVolumes(ctx, nodes, drives, podNames, podNSs, k8s.MaxThreadCount)
	if err != nil {
		return nil, err
	}

	volumeList := []types.Volume{}
	for result := range resultCh {
		if result.Err != nil {
			return volumeList, result.Err
		}
		volumeList = append(volumeList, result.Volume)
	}

	return volumeList, nil
}

// ProcessVolumes processes the volumes by applying the provided filter functions
func ProcessVolumes(
	ctx context.Context,
	resultCh <-chan ListVolumeResult,
	matchFunc func(*types.Volume) bool,
	applyFunc func(*types.Volume) error,
	processFunc func(context.Context, *types.Volume) error,
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
				volume := result.Volume
				oresult.Object = &volume
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
			return matchFunc(object.(*types.Volume))
		},
		func(object runtime.Object) error {
			return applyFunc(object.(*types.Volume))
		},
		func(ctx context.Context, object runtime.Object) error {
			return processFunc(ctx, object.(*types.Volume))
		},
		writer,
		dryRun,
	)
}

// VolumesListerWatcher is the lister watcher for volumes.
func VolumesListerWatcher(nodeID string) cache.ListerWatcher {
	labelSelector := ""
	if nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", types.NodeLabelKey, types.NewLabelValue(nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		client.RESTClient(),
		consts.VolumeResource,
		"",
		optionsModifier,
	)
}
