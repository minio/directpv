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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
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

// Lister is drive lister.
type Lister struct {
	nodes          []directpvtypes.LabelValue
	driveNames     []directpvtypes.LabelValue
	accessTiers    []directpvtypes.LabelValue
	statusList     []directpvtypes.DriveStatus
	driveIDs       []directpvtypes.DriveID
	maxObjects     int64
	ignoreNotFound bool
}

// NewLister creates new drive lister.
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

// AccessTierSelector adds filter listing by access-tiers.
func (lister *Lister) AccessTierSelector(accessTiers []directpvtypes.LabelValue) *Lister {
	lister.accessTiers = accessTiers
	return lister
}

// StatusSelector adds filter listing by drive status.
func (lister *Lister) StatusSelector(statusList []directpvtypes.DriveStatus) *Lister {
	lister.statusList = statusList
	return lister
}

// DriveIDSelector adds filter listing by drive IDs.
func (lister *Lister) DriveIDSelector(driveIDs []directpvtypes.DriveID) *Lister {
	lister.driveIDs = driveIDs
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

// List returns channel to loop through drive items.
func (lister *Lister) List(ctx context.Context) <-chan ListDriveResult {
	getOnly := len(lister.nodes) == 0 &&
		len(lister.driveNames) == 0 &&
		len(lister.accessTiers) == 0 &&
		len(lister.statusList) == 0 &&
		len(lister.driveIDs) != 0

	labelSelector := directpvtypes.ToLabelSelector(
		map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
			directpvtypes.NodeLabelKey:       lister.nodes,
			directpvtypes.DriveNameLabelKey:  lister.driveNames,
			directpvtypes.AccessTierLabelKey: lister.accessTiers,
		},
	)

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
				result, err := client.DriveClient().List(ctx, options)
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
			drive, err := client.DriveClient().Get(ctx, string(driveID), metav1.GetOptions{})
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
func (lister *Lister) Get(ctx context.Context) ([]types.Drive, error) {
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
