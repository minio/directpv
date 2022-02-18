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

package uevent

import (
	"time"

	"github.com/kubernetes/client-go/tools/cache"
)

type indexer struct {
	store cache.Store
}

func newIndexer(ctx context.Context, resyncPeriod time.Duration) *indexer {
	store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	lw := utils.DrivesListerWatcher(nodeID)
	reflector := cache.NewReflector(lw, &directcsi.DirectCSIDrive{}, store, resyncPeriod)

	go reflector.Run(ctx.Done())
	
	return &indexer{
		store: store,
	}
}

func (i *indexer) validateDevice(device *sys.Device) (bool, error) {
	
}

func (i *indexer) getDeviceCRD(device *sys.Device) (*directcsi.DirectCSIDrive, error) {

}
