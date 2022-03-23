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
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetRootBlockPath(t1 *testing.T) {

	testCases := []struct {
		name     string
		devName  string
		rootFile string
	}{
		{
			name:     "test1",
			devName:  "/dev/xvdb",
			rootFile: "/dev/xvdb",
		},
		{
			name:     "test2",
			devName:  "/dev/xvdb1",
			rootFile: "/dev/xvdb1",
		},
		{
			name:     "test3",
			devName:  "/var/lib/direct-csi/devices/xvdb",
			rootFile: "/dev/xvdb",
		},
		{
			name:     "test4",
			devName:  "/var/lib/direct-csi/devices/xvdb-part-3",
			rootFile: "/dev/xvdb3",
		},
		{
			name:     "test5",
			devName:  "/var/lib/direct-csi/devices/xvdb-part-15",
			rootFile: "/dev/xvdb15",
		},
		{
			name:     "test6",
			devName:  "/var/lib/direct-csi/devices/nvmen1p-part-4",
			rootFile: "/dev/nvmen1p4",
		},
		{
			name:     "test7",
			devName:  "/var/lib/direct-csi/devices/nvmen12p-part-11",
			rootFile: "/dev/nvmen12p11",
		},
		{
			name:     "test8",
			devName:  "/var/lib/direct-csi/devices/loop0",
			rootFile: "/dev/loop0",
		},
		{
			name:     "test9",
			devName:  "/var/lib/direct-csi/devices/loop-part-5",
			rootFile: "/dev/loop5",
		},
		{
			name:     "test10",
			devName:  "/var/lib/direct-csi/devices/loop-part-12",
			rootFile: "/dev/loop12",
		},
		{
			name:     "test11",
			devName:  "loop12",
			rootFile: "/dev/loop12",
		},
		{
			name:     "test12",
			devName:  "loop0",
			rootFile: "/dev/loop0",
		},
		{
			name:     "test13",
			devName:  "/var/lib/direct-csi/devices/nvmen-part-1-part-4",
			rootFile: "/dev/nvmen1p4",
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			rootFile := getRootBlockPath(tt.devName)
			if rootFile != tt.rootFile {
				t1.Errorf("Test case name %s: Expected root file = (%s) got: %s", tt.name, tt.rootFile, rootFile)
			}
		})
	}

}
func TestValidateUDevInfo(t1 *testing.T) {
	testCases := []struct {
		device         *sys.Device
		drives         []*directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{
			device: &sys.Device{
				Name:              "name",
				Major:             200,
				Minor:             2,
				Size:              uint64(5368709120),
				WWID:              "wwid",
				Model:             "model",
				Vendor:            "vendor",
				DMName:            "dmname",
				DMUUID:            "dmuuid",
				MDUUID:            "mduuid",
				PTUUID:            "ptuuid",
				PTType:            "gpt",
				PartUUID:          "partuuid",
				FSUUID:            "fsuuid",
				FSType:            "xfs",
				UeventSerial:      "ueventserial",
				UeventFSUUID:      "d9877501-e1b5-4bac-b73f-178b29974ed5",
				TotalCapacity:     uint64(5368709120),
				FreeCapacity:      uint64(5368709120),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						UeventSerial: "ueventserial",
						UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_2",
					},
					Status: directcsi.DirectCSIDriveStatus{
						UeventSerial:   "ueventserial",
						FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "sdb",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/sdb",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: true,
		},
		{
			device: &sys.Device{
				Name:         "sda", // changed
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        202, // changed
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        3, // changed
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid2", // changed
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model2", // changed
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor2", // changed
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor",
				DMName:       "dmname2", // changed
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid2", // changed
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid2", // changed
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid", // changed
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid2",
				PTUUID:       "parttableuuid2", // changed
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "name",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Serial:       "serial",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid2",
				PTUUID:       "parttableuuid",
				PTType:       "gpt", // changed
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/name",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "dos",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "sdb",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid2", // changed
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/sdb",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "sdb",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "ext4", // changed
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/sdb",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "sdb",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial2", // changed
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(0),
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/sdb",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
		{
			device: &sys.Device{
				Name:         "sdb",
				Major:        200,
				Minor:        2,
				WWID:         "wwid",
				Model:        "model",
				Vendor:       "vendor",
				DMName:       "dmname",
				DMUUID:       "dmuuid",
				MDUUID:       "mduuid",
				PTUUID:       "parttableuuid",
				PTType:       "gpt",
				PartUUID:     "partuuid",
				FSType:       "xfs",
				UeventSerial: "ueventserial",
				UeventFSUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
				Virtual:      false,
				Partition:    int(2), // changed
			},
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name: "test_drive_1",
					},
					Status: directcsi.DirectCSIDriveStatus{
						Path:          "/dev/sdb",
						MajorNumber:   uint32(200),
						MinorNumber:   uint32(2),
						Virtual:       false,
						Filesystem:    "xfs",
						PartitionNum:  0,
						WWID:          "wwid",
						ModelNumber:   "model",
						UeventSerial:  "ueventserial",
						Vendor:        "vendor",
						DMName:        "dmname",
						DMUUID:        "dmuuid",
						MDUUID:        "mduuid",
						PartTableUUID: "parttableuuid",
						PartTableType: "gpt",
						PartitionUUID: "partuuid",
						UeventFSUUID:  "d9877501-e1b5-4bac-b73f-178b29974ed5",
					},
				},
			},
			expectedResult: false,
		},
	}

	for i, testCase := range testCases {
		if testCase.expectedResult != ValidateUDevInfo(testCase.device, testCase.drives[0]) {
			t1.Errorf("Test case %d: Expected result = (%v) got: %v", i, testCase.expectedResult, !testCase.expectedResult)
		}
	}
}
