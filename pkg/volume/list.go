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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListVolumeResult denotes list of volume result.
type ListVolumeResult struct {
	Volume types.Volume
	Err    error
}

// Lister is volume lister.
type Lister struct {
	nodes          []directpvtypes.LabelValue
	driveNames     []directpvtypes.LabelValue
	driveIDs       []directpvtypes.LabelValue
	podNames       []directpvtypes.LabelValue
	podNSs         []directpvtypes.LabelValue
	statusList     []directpvtypes.VolumeStatus
	volumeNames    []string
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

// DriveNameSelector adds filter listing by drive names.
func (lister *Lister) DriveNameSelector(driveNames []directpvtypes.LabelValue) *Lister {
	lister.driveNames = driveNames
	return lister
}

// DriveIDSelector adds filter listing by drive IDs.
func (lister *Lister) DriveIDSelector(driveIDs []directpvtypes.LabelValue) *Lister {
	lister.driveIDs = driveIDs
	return lister
}

// PodNameSelector adds filter listing by pod names.
func (lister *Lister) PodNameSelector(podNames []directpvtypes.LabelValue) *Lister {
	lister.podNames = podNames
	return lister
}

// PodNSSelector adds filter listing by pod namespaces.
func (lister *Lister) PodNSSelector(podNSs []directpvtypes.LabelValue) *Lister {
	lister.podNSs = podNSs
	return lister
}

// StatusSelector adds filter listing by volume status.
func (lister *Lister) StatusSelector(statusList []directpvtypes.VolumeStatus) *Lister {
	lister.statusList = statusList
	return lister
}

// VolumeNameSelector adds filter listing by volume names.
func (lister *Lister) VolumeNameSelector(volumeNames []string) *Lister {
	lister.volumeNames = volumeNames
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *Lister) MaxObjects(n int64) *Lister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore drive not found error.
func (lister *Lister) IgnoreNotFound(b bool) *Lister {
	lister.ignoreNotFound = b
	return lister
}

// List returns channel to loop through volume items.
func (lister *Lister) List(ctx context.Context) <-chan ListVolumeResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.driveNames) == 0 &&
		len(lister.driveIDs) == 0 &&
		len(lister.podNames) == 0 &&
		len(lister.podNSs) == 0 &&
		len(lister.statusList) == 0 &&
		len(lister.volumeNames) != 0

	labelSelector := directpvtypes.ToLabelSelector(
		map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
			directpvtypes.NodeLabelKey:      lister.nodes,
			directpvtypes.DriveNameLabelKey: lister.driveNames,
			directpvtypes.DriveLabelKey:     lister.driveIDs,
			directpvtypes.PodNameLabelKey:   lister.podNames,
			directpvtypes.PodNSLabelKey:     lister.podNSs,
		},
	)

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
				result, err := client.VolumeClient().List(ctx, options)
				if err != nil {
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
			volume, err := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{})
			if err != nil {
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
func (lister *Lister) Get(ctx context.Context) ([]types.Volume, error) {
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
