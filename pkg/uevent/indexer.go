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
	"context"
	"errors"
	"time"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"k8s.io/client-go/tools/cache"
)

var (
	errNotDirectCSIDriveObject = errors.New("not a directcsidrive object")
)

type indexer struct {
	store  cache.Store
	nodeID string
}

func newIndexer(ctx context.Context, nodeID string, resyncPeriod time.Duration) *indexer {
	store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	lw := client.DrivesListerWatcher(nodeID)
	reflector := cache.NewReflector(lw, &directcsi.DirectCSIDrive{}, store, resyncPeriod)

	go reflector.Run(ctx.Done())

	return &indexer{
		store:  store,
		nodeID: nodeID,
	}
}

func (i *indexer) validateDevice(device *sys.Device) (bool, error) {
	if device.UeventFSUUID == "" {
		return false, nil
	}

	filteredDrives, err := i.filterDrivesByUEventFSUUID(device.UeventFSUUID)
	if err != nil {
		return false, err
	}

	if len(filteredDrives) != 1 {
		// To-Do: handle if more than one drive is found for a given path and node
		return false, nil
	}

	directCSIDrive := filteredDrives[0]

	if directCSIDrive.Status.Path != device.DevPath() {
		return false, nil
	}
	if directCSIDrive.Status.MajorNumber != uint32(device.Major) {
		return false, nil
	}
	if directCSIDrive.Status.MinorNumber != uint32(device.Minor) {
		return false, nil
	}
	if directCSIDrive.Status.Virtual != device.Virtual {
		return false, nil
	}
	if directCSIDrive.Status.PartitionNum != device.Partition {
		return false, nil
	}
	if directCSIDrive.Status.WWID != device.WWID {
		return false, nil
	}
	if directCSIDrive.Status.ModelNumber != device.Model {
		return false, nil
	}
	if directCSIDrive.Status.UeventSerial != device.UeventSerial {
		return false, nil
	}
	if directCSIDrive.Status.Vendor != device.Vendor {
		return false, nil
	}
	if directCSIDrive.Status.DMName != device.DMName {
		return false, nil
	}
	if directCSIDrive.Status.DMUUID != device.DMUUID {
		return false, nil
	}
	if directCSIDrive.Status.MDUUID != device.MDUUID {
		return false, nil
	}
	if directCSIDrive.Status.PartTableUUID != device.PTUUID {
		return false, nil
	}
	if directCSIDrive.Status.PartTableType != device.PTType {
		return false, nil
	}
	if directCSIDrive.Status.PartitionUUID != device.PartUUID {
		return false, nil
	}
	if directCSIDrive.Status.Filesystem != device.FSType {
		return false, nil
	}

	return true, nil
}

func (i *indexer) filterDrivesByUEventFSUUID(fsuuid string) ([]*directcsi.DirectCSIDrive, error) {
	objects := i.store.List()
	filteredDrives := []*directcsi.DirectCSIDrive{}
	for _, obj := range objects {
		directCSIDrive, ok := obj.(*directcsi.DirectCSIDrive)
		if !ok {
			return nil, errNotDirectCSIDriveObject
		}
		if directCSIDrive.Status.NodeName != i.nodeID {
			continue
		}
		if directCSIDrive.Status.UeventFSUUID != fsuuid {
			continue
		}
		filteredDrives = append(filteredDrives, directCSIDrive)
	}
	return filteredDrives, nil
}

func (i *indexer) listDrives() ([]*directcsi.DirectCSIDrive, error) {
	objects := i.store.List()
	drives := []*directcsi.DirectCSIDrive{}
	for _, obj := range objects {
		directCSIDrive, ok := obj.(*directcsi.DirectCSIDrive)
		if !ok {
			return nil, errNotDirectCSIDriveObject
		}
		if directCSIDrive.Status.NodeName != i.nodeID {
			continue
		}
		drives = append(drives, directCSIDrive)
	}
	return drives, nil
}
