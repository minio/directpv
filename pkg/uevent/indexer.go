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
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"k8s.io/client-go/tools/cache"
)

var (
	errNotDirectCSIDriveObject = errors.New("not a directcsidrive object")
	errNoMatchFound            = errors.New("no matching drive found")
)

type indexer struct {
	store  cache.Store
	nodeID string
}

func newIndexer(ctx context.Context, nodeID string, resyncPeriod time.Duration) *indexer {
	store := cache.NewStore(directCSIDriveKeyFunc)

	lw := client.DrivesListerWatcher(nodeID)
	reflector := cache.NewReflector(lw, &directcsi.DirectCSIDrive{}, store, resyncPeriod)

	go reflector.Run(ctx.Done())

	return &indexer{
		store:  store,
		nodeID: nodeID,
	}
}

func directCSIDriveKeyFunc(obj interface{}) (string, error) {
	directCSIDrive, ok := obj.(*directcsi.DirectCSIDrive)
	if !ok {
		return "", errNotDirectCSIDriveObject
	}
	return getDirectCSIDriveKey(directCSIDrive), nil
}

func getDirectCSIDriveKey(directCSIDrive *directcsi.DirectCSIDrive) string {
	data := []byte(
		strings.Join(
			[]string{
				directCSIDrive.Status.NodeName,
				directCSIDrive.Status.Path,
				strconv.FormatUint(uint64(directCSIDrive.Status.MajorNumber), 10),
				strconv.FormatUint(uint64(directCSIDrive.Status.MinorNumber), 10),
				strconv.Itoa(directCSIDrive.Status.PartitionNum),
				directCSIDrive.Status.WWID,
				directCSIDrive.Status.ModelNumber,
				directCSIDrive.Status.UeventSerial,
				directCSIDrive.Status.Vendor,
				directCSIDrive.Status.DMName,
				directCSIDrive.Status.DMUUID,
				directCSIDrive.Status.MDUUID,
				directCSIDrive.Status.PartTableUUID,
				directCSIDrive.Status.PartTableType,
				directCSIDrive.Status.PartitionUUID,
				directCSIDrive.Status.Filesystem,
				directCSIDrive.Status.UeventFSUUID,
			},
			"-",
		),
	)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (i *indexer) deriveDirectCSIDriveKey(device *sys.Device) string {
	data := []byte(
		strings.Join(
			[]string{
				i.nodeID,
				device.DevPath(),
				strconv.Itoa(device.Major),
				strconv.Itoa(device.Minor),
				strconv.Itoa(device.Partition),
				device.WWID,
				device.Model,
				device.UeventSerial,
				device.Vendor,
				device.DMName,
				device.DMUUID,
				device.MDUUID,
				device.PTUUID,
				device.PTType,
				device.PartUUID,
				device.FSType,
				device.UeventFSUUID,
			},
			"-",
		),
	)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (i *indexer) validateDevice(device *sys.Device) (bool, error) {
	key := i.deriveDirectCSIDriveKey(device)
	_, exists, err := i.store.GetByKey(key)
	return exists, err
}

func (i *indexer) getMatchingDrive(device *sys.Device) (*directcsi.DirectCSIDrive, error) {
	filteredDrives, err := i.filterDrivesByPath(device.DevPath())
	if err != nil {
		return nil, err
	}
	// To-Do/Fix-me: run matching algorithm to find matching drive
	return filteredDrives[0], errNoMatchFound
}

func (i *indexer) filterDrivesByPath(path string) ([]*directcsi.DirectCSIDrive, error) {
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
		if directCSIDrive.Status.Path != path {
			continue
		}
		filteredDrives = append(filteredDrives, directCSIDrive)
	}
	return filteredDrives, nil
}
