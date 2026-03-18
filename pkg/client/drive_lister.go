// This file is part of MinIO DirectPV
// Copyright (c) 2021-2024 MinIO, Inc.
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

// ListDriveResult denotes list of drive result.
type ListDriveResult struct {
	Drive types.Drive
	Err   error
}

// DriveLister is lister wrapper for DirectPVDrive listing.
type DriveLister struct {
	nodes          []directpvtypes.LabelValue
	driveNames     []directpvtypes.LabelValue
	accessTiers    []directpvtypes.LabelValue
	statusList     []directpvtypes.DriveStatus
	driveIDs       []directpvtypes.DriveID
	labels         map[directpvtypes.LabelKey]directpvtypes.LabelValue
	maxObjects     int64
	ignoreNotFound bool
	driveClient    types.LatestDriveInterface
}

// NewDriveLister creates new drive lister.
func (c Client) NewDriveLister() *DriveLister {
	return &DriveLister{
		maxObjects:  k8s.MaxThreadCount,
		driveClient: c.Drive(),
	}
}

// NodeSelector adds filter listing by nodes.
func (lister *DriveLister) NodeSelector(nodes []directpvtypes.LabelValue) *DriveLister {
	lister.nodes = nodes
	return lister
}

// DriveNameSelector adds filter listing by drive names.
func (lister *DriveLister) DriveNameSelector(driveNames []directpvtypes.LabelValue) *DriveLister {
	lister.driveNames = driveNames
	return lister
}

// StatusSelector adds filter listing by drive status.
func (lister *DriveLister) StatusSelector(statusList []directpvtypes.DriveStatus) *DriveLister {
	lister.statusList = statusList
	return lister
}

// DriveIDSelector adds filter listing by drive IDs.
func (lister *DriveLister) DriveIDSelector(driveIDs []directpvtypes.DriveID) *DriveLister {
	lister.driveIDs = driveIDs
	return lister
}

// LabelSelector adds filter listing by labels.
func (lister *DriveLister) LabelSelector(labels map[directpvtypes.LabelKey]directpvtypes.LabelValue) *DriveLister {
	lister.labels = labels
	return lister
}

// MaxObjects controls number of items to be fetched in every iteration.
func (lister *DriveLister) MaxObjects(n int64) *DriveLister {
	lister.maxObjects = n
	return lister
}

// IgnoreNotFound controls listing to ignore drive not found error.
func (lister *DriveLister) IgnoreNotFound(b bool) *DriveLister {
	lister.ignoreNotFound = b
	return lister
}

// List returns channel to loop through drive items.
func (lister *DriveLister) List(ctx context.Context) <-chan ListDriveResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.driveNames) == 0 &&
		len(lister.accessTiers) == 0 &&
		len(lister.statusList) == 0 &&
		len(lister.labels) == 0 &&
		len(lister.driveIDs) != 0

	labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
		directpvtypes.NodeLabelKey:       lister.nodes,
		directpvtypes.DriveNameLabelKey:  lister.driveNames,
		directpvtypes.AccessTierLabelKey: lister.accessTiers,
	}
	for k, v := range lister.labels {
		labelMap[k] = []directpvtypes.LabelValue{v}
	}
	labelSelector := directpvtypes.ToLabelSelector(labelMap)

	resultCh := make(chan ListDriveResult)
	go func() {
		defer close(resultCh)

		send := func(result ListDriveResult) bool {
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
				result, err := lister.driveClient.List(ctx, options)
				if err != nil {
					if apierrors.IsNotFound(err) && lister.ignoreNotFound {
						break
					}
					send(ListDriveResult{Err: err})
					return
				}

				for _, item := range result.Items {
					var found bool
					var values []directpvtypes.DriveID
					for i := range lister.driveIDs {
						if lister.driveIDs[i] == item.GetDriveID() {
							found = true
						} else {
							values = append(values, lister.driveIDs[i])
						}
					}
					lister.driveIDs = values

					if found || len(lister.statusList) == 0 || utils.Contains(lister.statusList, item.Status.Status) {
						if !send(ListDriveResult{Drive: item}) {
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

		for _, driveID := range lister.driveIDs {
			drive, err := lister.driveClient.Get(ctx, string(driveID), metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) && lister.ignoreNotFound {
					continue
				}

				send(ListDriveResult{Err: err})
				return
			}

			if !send(ListDriveResult{Drive: *drive}) {
				return
			}
		}
	}()

	return resultCh
}

// Get returns list of drives.
func (lister *DriveLister) Get(ctx context.Context) ([]types.Drive, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	driveList := []types.Drive{}
	for result := range lister.List(ctx) {
		if result.Err != nil {
			return driveList, result.Err
		}
		driveList = append(driveList, result.Drive)
	}

	return driveList, nil
}
