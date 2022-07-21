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
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeDriveEventHander struct {
	nodeID  string
	added   []string
	updated []string
	deleted []string
}

func (d *fakeDriveEventHander) Add(ctx context.Context, device *sys.Device) error {
	d.added = append(d.added, device.Name)
	return nil
}

func (d *fakeDriveEventHander) Update(ctx context.Context, device *sys.Device, drive *directcsi.DirectCSIDrive) error {
	d.updated = append(d.updated, device.Name)
	return nil
}

func (d *fakeDriveEventHander) Remove(ctx context.Context, drive *directcsi.DirectCSIDrive) (err error) {
	d.deleted = append(d.deleted, filepath.Base(drive.Status.Path))
	return nil
}

func TestSyncDevices(t *testing.T) {
	createTestDrive := func(name, path, wwid, fsuuid, fsType string, partitionNum int, driveStatus directcsi.DriveStatus, majNum, minNum uint32) *directcsi.DirectCSIDrive {
		csiDrive := &directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceNone,
			},
			Status: directcsi.DirectCSIDriveStatus{
				NodeName:       "test-node",
				Path:           path,
				MajorNumber:    majNum,
				MinorNumber:    minNum,
				PartitionNum:   partitionNum,
				WWID:           wwid,
				FilesystemUUID: fsuuid,
				Filesystem:     fsType,
				DriveStatus:    driveStatus,
			},
		}
		utils.UpdateLabels(csiDrive, map[utils.LabelKey]utils.LabelValue{
			utils.NodeLabelKey: utils.NewLabelValue(csiDrive.Status.NodeName),
		})
		if utils.IsManagedDrive(csiDrive) {
			csiDrive.Finalizers = []string{directcsi.DirectCSIDriveFinalizerDataProtection}
		}
		return csiDrive
	}

	createTestDevice := func(name, wwid, fsuuid, fsType string, partitionNum, majNum, minNum int) *sys.Device {
		return &sys.Device{
			Name:      name,
			Major:     majNum,
			Minor:     minNum,
			Partition: partitionNum,
			WWID:      wwid,
			FSUUID:    fsuuid,
			FSType:    fsType,
		}
	}

	testCases := []struct {
		drives             []*directcsi.DirectCSIDrive
		devices            []*sys.Device
		addedDeviceNames   []string
		updatedDeviceNames []string
		deletedDeviceNames []string
	}{
		// matching drives with perfect match
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"wwid-e",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"wwid-f",
					"66666666-6666-6666-6666-666666666666",
					"ext4",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "44444444-4444-4444-4444-444444444444", "xfs", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
				createTestDevice("sdf", "wwid-f", "66666666-6666-6666-6666-666666666666", "ext4", 0, 202, 6),
			},
			addedDeviceNames:   []string{},
			updatedDeviceNames: []string{},
			deletedDeviceNames: []string{},
		},
		// matching drives and updating changes
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"wwid-e",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"wwid-f",
					"66666666-6666-6666-6666-666666666666",
					"ext4",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 22),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 33),
				createTestDevice("sdd", "wwid-d", "44444444-4444-4444-4444-444444444444", "xfs", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
				createTestDevice("sdf", "wwid-f", "66666666-6666-6666-6666-666666666666", "ext4", 0, 202, 66),
			},
			addedDeviceNames:   []string{},
			updatedDeviceNames: []string{"sdb", "sdc", "sdf"},
			deletedDeviceNames: []string{},
		},
		// matching unidentified drives
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "", "", "", 0, 202, 4),
				createTestDevice("sde", "", "", "", 0, 202, 5),
				createTestDevice("sdf", "", "", "", 0, 202, 6),
			},
			addedDeviceNames:   []string{},
			updatedDeviceNames: []string{},
			deletedDeviceNames: []string{},
		},
		// matching unidentified drives and updating changes
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "", "", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
				createTestDevice("sdf", "wwid-f", "", "", 0, 202, 6),
			},
			addedDeviceNames:   []string{},
			updatedDeviceNames: []string{"sdd", "sde", "sdf"},
			deletedDeviceNames: []string{},
		},
		// detecting detached drives
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"wwid-e",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"wwid-f",
					"66666666-6666-6666-6666-666666666666",
					"ext4",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "44444444-4444-4444-4444-444444444444", "xfs", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
			},
			addedDeviceNames:   []string{},
			updatedDeviceNames: []string{},
			deletedDeviceNames: []string{"sdb", "sdf"},
		},
		// detecting attached drives
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "44444444-4444-4444-4444-444444444444", "xfs", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
				createTestDevice("sdf", "wwid-f", "66666666-6666-6666-6666-666666666666", "ext4", 0, 202, 6),
			},
			addedDeviceNames:   []string{"sde", "sdf"},
			updatedDeviceNames: []string{},
			deletedDeviceNames: []string{},
		},
		// detecting both attached and detached drives
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"wwid-e",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 55),
				createTestDevice("sdf", "wwid-f", "66666666-6666-6666-6666-666666666666", "ext4", 0, 202, 6),
			},
			addedDeviceNames:   []string{"sdf"},
			updatedDeviceNames: []string{"sde"},
			deletedDeviceNames: []string{"sdd"},
		},
		// detecting attached and detached unidentified drives
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-111111111111",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-333333333333",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"",
					"",
					"",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "", "", 0, 202, 4),
				createTestDevice("sde", "", "", "", 0, 202, 5),
			},
			addedDeviceNames:   []string{"sde"},
			updatedDeviceNames: []string{"sdd"},
			deletedDeviceNames: []string{"sdf"},
		},
		// detecting corrupt inuse drives (corrupted drives will be added freshly)
		{
			drives: []*directcsi.DirectCSIDrive{
				createTestDrive(
					"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
					"/dev/sda",
					"wwid-a",
					"11111111-1111-1111-1111-1111111111XX",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(1),
				),
				createTestDrive(
					"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
					"/dev/sdb",
					"wwid-b",
					"22222222-2222-2222-2222-222222222222",
					"xfs",
					0,
					directcsi.DriveStatusInUse,
					uint32(202),
					uint32(2),
				),
				createTestDrive(
					"cccccccc-cccc-cccc-cccc-cccccccccccc",
					"/dev/sdc",
					"wwid-c",
					"33333333-3333-3333-3333-3333333333XX",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(3),
				),
				createTestDrive(
					"dddddddd-dddd-dddd-dddd-dddddddddddd",
					"/dev/sdd",
					"wwid-d",
					"44444444-4444-4444-4444-444444444444",
					"xfs",
					0,
					directcsi.DriveStatusReady,
					uint32(202),
					uint32(4),
				),
				createTestDrive(
					"eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
					"/dev/sde",
					"wwid-e",
					"",
					"",
					0,
					directcsi.DriveStatusAvailable,
					uint32(202),
					uint32(5),
				),
				createTestDrive(
					"ffffffff-ffff-ffff-ffff-ffffffffffff",
					"/dev/sdf",
					"wwid-f",
					"66666666-6666-6666-6666-666666666666",
					"ext4",
					0,
					directcsi.DriveStatusUnavailable,
					uint32(202),
					uint32(6),
				),
			},
			devices: []*sys.Device{
				createTestDevice("sda", "wwid-a", "11111111-1111-1111-1111-111111111111", "xfs", 0, 202, 1),
				createTestDevice("sdb", "wwid-b", "22222222-2222-2222-2222-222222222222", "xfs", 0, 202, 2),
				createTestDevice("sdc", "wwid-c", "33333333-3333-3333-3333-333333333333", "xfs", 0, 202, 3),
				createTestDevice("sdd", "wwid-d", "44444444-4444-4444-4444-444444444444", "xfs", 0, 202, 4),
				createTestDevice("sde", "wwid-e", "", "", 0, 202, 5),
				createTestDevice("sdf", "wwid-f", "66666666-6666-6666-6666-666666666666", "ext4", 0, 202, 6),
			},
			addedDeviceNames:   []string{"sda", "sdc"},
			updatedDeviceNames: []string{},
			deletedDeviceNames: []string{"sda", "sdc"},
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	for i, testCase := range testCases {
		fakeIndexer := createFakeIndexer()
		for _, drive := range testCase.drives {
			if err := fakeIndexer.store.Add(drive); err != nil {
				t.Errorf("error while adding objects to store: %v", err)
			}
		}
		driveHandler := &fakeDriveEventHander{"test-node", []string{}, []string{}, []string{}}
		fakeListener := &listener{
			closeCh:    make(chan struct{}),
			eventQueue: newEventQueue(),
			nodeID:     "test-node",
			handler:    driveHandler,
			indexer:    fakeIndexer,
		}
		if err := fakeListener.syncDevices(ctx, testCase.devices); err != nil {
			t.Fatalf("case: %d error while syncing devices: %v", i, err)
		}
		sort.Strings(driveHandler.added)
		sort.Strings(testCase.addedDeviceNames)
		if !reflect.DeepEqual(driveHandler.added, testCase.addedDeviceNames) {
			t.Errorf("case: %d expected addedDeviceList: %v but got %v", i, testCase.addedDeviceNames, driveHandler.added)
		}
		sort.Strings(driveHandler.updated)
		sort.Strings(testCase.updatedDeviceNames)
		if !reflect.DeepEqual(driveHandler.updated, testCase.updatedDeviceNames) {
			t.Errorf("case: %d expected updatedDeviceList: %v but got %v", i, testCase.updatedDeviceNames, driveHandler.updated)
		}
		sort.Strings(driveHandler.deleted)
		sort.Strings(testCase.deletedDeviceNames)
		if !reflect.DeepEqual(driveHandler.deleted, testCase.deletedDeviceNames) {
			t.Errorf("case: %d expected deletedDeviceList: %v but got %v", i, testCase.deletedDeviceNames, driveHandler.deleted)
		}
	}
}
