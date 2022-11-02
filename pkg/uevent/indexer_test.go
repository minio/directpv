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
	"reflect"
	"sort"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	mb20 = 20 * 1024 * 1024
)

func createFakeIndexer() *indexer {
	store := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
	return &indexer{
		store:  store,
		nodeID: "test-node",
	}
}

func createTestDrive(node, drive, fsuuid string) *directcsi.DirectCSIDrive {
	return &directcsi.DirectCSIDrive{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name: drive,
			Finalizers: []string{
				string(directcsi.DirectCSIDriveFinalizerDataProtection),
			},
		},
		Status: directcsi.DirectCSIDriveStatus{
			NodeName:          node,
			Filesystem:        "xfs",
			DriveStatus:       directcsi.DriveStatusReady,
			FreeCapacity:      mb20,
			AllocatedCapacity: int64(0),
			TotalCapacity:     mb20,
			FilesystemUUID:    fsuuid,
		},
	}
}

func TestFilterDrivesByUEventFSUUID(t *testing.T) {
	filterOnFSuuid := "b9475609-e1b5-4986-vs33-178131rdes97"
	d1 := createTestDrive("test-node", "D1", "d9877501-e1b5-4bac-b73f-178b29974ed5")
	d2 := createTestDrive("test-node", "D2", filterOnFSuuid)
	indexer := createFakeIndexer()
	if err := indexer.store.Add(d1); err != nil {
		t.Errorf("error while adding objects to store: %v", err)
	}
	if err := indexer.store.Add(d2); err != nil {
		t.Errorf("error while adding objects to store: %v", err)
	}
	filteredDrive, err := indexer.filterDrivesByUEventFSUUID(filterOnFSuuid)
	if err != nil {
		t.Errorf("")
	}
	for _, val := range filteredDrive {
		if !reflect.DeepEqual(val.Status.FilesystemUUID, filterOnFSuuid) {
			t.Errorf("expected drive with FSUUID: %v but got: %v", filterOnFSuuid, val.Status.FilesystemUUID)
		}
	}
}

func TestListDrives(t *testing.T) {
	d1 := createTestDrive("test-node", "D1", "d9877501-e1b5-4bac-b73f-178b29974ed5")
	d2 := createTestDrive("test-node", "D2", "b9475609-e1b5-4986-vs33-178131rdes9")
	var drives []*directcsi.DirectCSIDrive
	drives = append(drives, d1, d2)
	indexer := createFakeIndexer()
	if err := indexer.store.Add(d1); err != nil {
		t.Errorf("error while adding objects to store: %v", err)
	}
	if err := indexer.store.Add(d2); err != nil {
		t.Errorf("error while adding objects to store: %v", err)
	}
	managedDrives, nonManagedDrives, _ := indexer.listDrives()
	listedDrives := append(managedDrives, nonManagedDrives...)
	sort.Slice(drives, func(p, q int) bool {
		return drives[p].Status.FilesystemUUID < drives[q].Status.FilesystemUUID
	})
	sort.Slice(listedDrives, func(p, q int) bool {
		return listedDrives[p].Status.FilesystemUUID < listedDrives[q].Status.FilesystemUUID
	})
	if !reflect.DeepEqual(drives, listedDrives) {
		t.Errorf("expected drive slice: %v but got: %v", drives, listedDrives)
	}
}
