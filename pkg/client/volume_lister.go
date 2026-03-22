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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListVolumeResult denotes list of volume result.
type ListVolumeResult struct {
	Volume types.Volume
	Err    error
}

// VolumeLister is volume lister.
type VolumeLister struct {
	nodes          []directpvtypes.LabelValue
	driveNames     []directpvtypes.LabelValue
	driveIDs       []directpvtypes.LabelValue
	podNames       []directpvtypes.LabelValue
	podNSs         []directpvtypes.LabelValue
	statusList     []directpvtypes.VolumeStatus
	volumeNames    []string
	labels         map[directpvtypes.LabelKey]directpvtypes.LabelValue
	maxObjects     int64
	ignoreNotFound bool
	volumeClient   types.LatestVolumeInterface
}

// NewVolumeLister creates new volume lister.
func (c Client) NewVolumeLister() *VolumeLister {
	return &VolumeLister{
		maxObjects:   k8s.MaxThreadCount,
		volumeClient: c.Volume(),
	}
}

// NodeSelector adds filter listing by nodes.
func (lister *VolumeLister) NodeSelector(nodes []directpvtypes.LabelValue) *VolumeLister {
	lister.nodes = nodes
	return lister
}

// DriveNameSelector adds filter listing by drive names.
func (lister *VolumeLister) DriveNameSelector(driveNames []directpvtypes.LabelValue) *VolumeLister {
	lister.driveNames = driveNames
	return lister
}

// DriveIDSelector adds filter listing by drive IDs.
func (lister *VolumeLister) DriveIDSelector(driveIDs []directpvtypes.LabelValue) *VolumeLister {
	lister.driveIDs = driveIDs
	return lister
}

// PodNameSelector adds filter listing by pod names.
func (lister *VolumeLister) PodNameSelector(podNames []directpvtypes.LabelValue) *VolumeLister {
	lister.podNames = podNames
	return lister
}

// PodNSSelector adds filter listing by pod namespaces.
func (lister *VolumeLister) PodNSSelector(podNSs []directpvtypes.LabelValue) *VolumeLister {
	lister.podNSs = podNSs
	return lister
}

// StatusSelector adds filter listing by volume status.
func (lister *VolumeLister) StatusSelector(statusList []directpvtypes.VolumeStatus) *VolumeLister {
	lister.statusList = statusList
	return lister
}

// VolumeNameSelector adds filter listing by volume names.
func (lister *VolumeLister) VolumeNameSelector(volumeNames []string) *VolumeLister {
	lister.volumeNames = volumeNames
	return lister
}

// LabelSelector adds filter listing by labels.
func (lister *VolumeLister) LabelSelector(labels map[directpvtypes.LabelKey]directpvtypes.LabelValue) *VolumeLister {
	lister.labels = labels
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *VolumeLister) MaxObjects(n int64) *VolumeLister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore drive not found error.
func (lister *VolumeLister) IgnoreNotFound(b bool) *VolumeLister {
	lister.ignoreNotFound = b
	return lister
}

// List returns channel to loop through volume items.
func (lister *VolumeLister) List(ctx context.Context) <-chan ListVolumeResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.driveNames) == 0 &&
		len(lister.driveIDs) == 0 &&
		len(lister.podNames) == 0 &&
		len(lister.podNSs) == 0 &&
		len(lister.statusList) == 0 &&
		len(lister.labels) == 0 &&
		len(lister.volumeNames) != 0

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey:      lister.nodes,
		directpvtypes.DriveNameLabelKey: lister.driveNames,
		directpvtypes.DriveLabelKey:     lister.driveIDs,
		directpvtypes.PodNameLabelKey:   lister.podNames,
		directpvtypes.PodNSLabelKey:     lister.podNSs,
	}
	for k, v := range lister.labels {
		labelMap[k] = []directpvtypes.LabelValue{v}
	}
	labelSelector := directpvtypes.ToLabelSelector(labelMap)

	resultCh := make(chan ListVolumeResult)
	go func() {
		defer close(resultCh)

		send := func(result ListVolumeResult) bool {
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
				result, err := lister.volumeClient.List(ctx, options)
				if err != nil {
					if apierrors.IsNotFound(err) && lister.ignoreNotFound {
						break
					}

					send(ListVolumeResult{Err: err})
					return
				}

				for _, item := range result.Items {
					var found bool
					var values []string
					for i := range lister.volumeNames {
						if lister.volumeNames[i] == item.Name {
							found = true
						} else {
							values = append(values, lister.volumeNames[i])
						}
					}
					lister.volumeNames = values

					if found || len(lister.statusList) == 0 || utils.Contains(lister.statusList, item.Status.Status) {
						if !send(ListVolumeResult{Volume: item}) {
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

		for _, volumeName := range lister.volumeNames {
			volume, err := lister.volumeClient.Get(ctx, volumeName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) && lister.ignoreNotFound {
					continue
				}

				send(ListVolumeResult{Err: err})
				return
			}
			if !send(ListVolumeResult{Volume: *volume}) {
				return
			}
		}
	}()

	return resultCh
}

// Get returns list of volumes.
func (lister *VolumeLister) Get(ctx context.Context) ([]types.Volume, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	volumeList := []types.Volume{}
	for result := range lister.List(ctx) {
		if result.Err != nil {
			return volumeList, result.Err
		}
		volumeList = append(volumeList, result.Volume)
	}

	return volumeList, nil
}
