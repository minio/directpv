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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetMatchedDevices(t *testing.T) {
	testCases := []struct {
		drive                    *directcsi.DirectCSIDrive
		devices                  []*sys.Device
		expectedMatchedDevices   []*sys.Device
		expectedUnmatchedDevices []*sys.Device
	}{
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			devices: []*sys.Device{
				{
					FSUUID: "xxxxx-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "yyyyy-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedUnmatchedDevices: []*sys.Device{
				{
					FSUUID: "xxxxx-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "yyyyy-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			devices: []*sys.Device{
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "yyyyy-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedMatchedDevices: []*sys.Device{
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedUnmatchedDevices: []*sys.Device{
				{
					FSUUID: "yyyyy-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			devices: []*sys.Device{
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedMatchedDevices: []*sys.Device{
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
				{
					FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedUnmatchedDevices: []*sys.Device{
				{
					FSUUID: "zzzzz-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
		},
	}

	for i, testCase := range testCases {
		matchedDevices, unmatchedDevices := getMatchedDevices(testCase.drive, testCase.devices, func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
			return drive.Status.FilesystemUUID == device.FSUUID
		})
		sort.Slice(matchedDevices, func(p, q int) bool {
			return matchedDevices[p].FSUUID < matchedDevices[q].FSUUID
		})
		sort.Slice(unmatchedDevices, func(p, q int) bool {
			return unmatchedDevices[p].FSUUID < unmatchedDevices[q].FSUUID
		})
		sort.Slice(testCase.expectedMatchedDevices, func(p, q int) bool {
			return testCase.expectedMatchedDevices[p].FSUUID < testCase.expectedMatchedDevices[q].FSUUID
		})
		sort.Slice(testCase.expectedUnmatchedDevices, func(p, q int) bool {
			return testCase.expectedUnmatchedDevices[p].FSUUID < testCase.expectedUnmatchedDevices[q].FSUUID
		})
		if !reflect.DeepEqual(testCase.expectedMatchedDevices, matchedDevices) {
			t.Errorf("case: %d expected matchedDevices: %v but got %v", i, testCase.expectedMatchedDevices, matchedDevices)
		}
		if !reflect.DeepEqual(testCase.expectedUnmatchedDevices, unmatchedDevices) {
			t.Errorf("case: %d expected unmatchedDevices: %v but got %v", i, testCase.expectedUnmatchedDevices, unmatchedDevices)
		}
	}
}

func TestGetMatchedDrives(t *testing.T) {
	testCases := []struct {
		drives                []*directcsi.DirectCSIDrive
		device                *sys.Device
		expectedMatchedDrives []*directcsi.DirectCSIDrive
	}{
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "lasbfqkp-0d9d-4e6e-a766-c79ac18b7ea6",
			},
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedMatchedDrives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedMatchedDrives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
		},
	}

	for i, testCase := range testCases {
		matchedDrives := getMatchedDrives(testCase.drives, testCase.device, func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
			return drive.Status.FilesystemUUID == device.FSUUID
		})
		sort.Slice(matchedDrives, func(p, q int) bool {
			return matchedDrives[p].Status.FilesystemUUID < matchedDrives[q].Status.FilesystemUUID
		})
		if !reflect.DeepEqual(testCase.expectedMatchedDrives, matchedDrives) {
			t.Errorf("case: %d expected matchedDrives: %v but got %v", i, testCase.expectedMatchedDrives, matchedDrives)
		}
	}
}

func TestFSMatcher(t *testing.T) {
	testCases := []struct {
		drive          *directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Filesystem:     "xfs",
				},
			},
			device: &sys.Device{
				FSType: "xfs",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: true,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Filesystem: "xfs",
				},
			},
			device: &sys.Device{
				FSType: "xfs",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			device: &sys.Device{
				FSType: "xfs",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{},
			},
			device: &sys.Device{
				FSType: "xfs",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Filesystem:     "xfs",
				},
			},
			device: &sys.Device{
				FSType: "ext4",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Filesystem:     "xfs",
				},
			},
			device: &sys.Device{
				FSType: "xfs",
				FSUUID: "xxxxxxx-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Filesystem:     "xfs",
				},
			},
			device: &sys.Device{
				FSType: "ext4",
				FSUUID: "xxxxxxxx-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedResult: false,
		},
	}

	for i, testCase := range testCases {
		result := fsMatcher(testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Errorf("case %d expected result: %v but got %v", i, testCase.expectedResult, result)
		}
	}
}

func TestConclusiveMatcher(t *testing.T) {
	testCases := []struct {
		drive          *directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{},
			},
			device:         &sys.Device{},
			expectedResult: false,
		},
		// PartitionNumber
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(0),
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		// WWID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					WWID:         "wwid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					WWID:         "naa.6002248032bf1752a69bdaee7b0ceb33",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "0x6002248032bf1752a69bdaee7b0ceb33",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					WWID:         "wwid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// UeventSerial
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					UeventSerial: "ueventserial-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					UeventSerial: "ueventserial-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// SerialNumberLong
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:     int(1),
					SerialNumberLong: "seriallong-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:     int(1),
					SerialNumberLong: "seriallong-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// DMUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					DMUUID:       "dmuuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					DMUUID:       "dmuuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// MDUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					MDUUID:       "mduuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					MDUUID:       "mduuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// ModelNumber and Vendor
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					ModelNumber:  "model-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					ModelNumber:  "model-xxx",
					Vendor:       "vendor-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					ModelNumber:  "model-xxx",
					Vendor:       "vendor-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		// PartitionUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:  int(1),
					ModelNumber:   "model-xxx",
					Vendor:        "vendor-xxx",
					PartitionUUID: "partuuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:  int(1),
					ModelNumber:   "model-xxx",
					Vendor:        "vendor-xxx",
					PartitionUUID: "PARTUUID-XXX",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:  int(1),
					ModelNumber:   "model-xxx",
					Vendor:        "vendor-xxx",
					PartitionUUID: "partuuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// PartTableUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:  int(1),
					ModelNumber:   "model-xxx",
					Vendor:        "vendor-xxx",
					PartTableUUID: "ptuuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:  int(1),
					ModelNumber:   "model-xxx",
					Vendor:        "vendor-xxx",
					PartTableUUID: "ptuuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// UeventFSUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					ModelNumber:  "model-xxx",
					Vendor:       "vendor-xxx",
					UeventFSUUID: "ueventfsuuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum: int(1),
					ModelNumber:  "model-xxx",
					Vendor:       "vendor-xxx",
					UeventFSUUID: "ueventfsuuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
		// FilesystemUUID
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:   int(1),
					ModelNumber:    "model-xxx",
					Vendor:         "vendor-xxx",
					FilesystemUUID: "fsuuid-yyy",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					PartitionNum:   int(1),
					ModelNumber:    "model-xxx",
					Vendor:         "vendor-xxx",
					FilesystemUUID: "fsuuid-xxx",
				},
			},
			device: &sys.Device{
				Partition:    1,
				WWID:         "wwid-xxx",
				UeventSerial: "ueventserial-xxx",
				SerialLong:   "seriallong-xxx",
				DMUUID:       "dmuuid-xxx",
				MDUUID:       "mduuid-xxx",
				Model:        "model-xxx",
				Vendor:       "vendor-xxx",
				PartUUID:     "partuuid-xxx",
				PTUUID:       "ptuuid-xxx",
				UeventFSUUID: "ueventfsuuid-xxx",
				FSUUID:       "fsuuid-xxx",
			},
			expectedResult: true,
		},
	}
	for i, testCase := range testCases {
		result := conclusiveMatcher(testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Errorf("case %d expected result: %v but got %v", i, testCase.expectedResult, result)
		}
	}
}

func TestNonConclusiveMatcher(t *testing.T) {
	testCases := []struct {
		drive          *directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sda",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
					PCIPath:     "pcipath-xxx",
				},
			},
			device: &sys.Device{
				Name:    "sda",
				Major:   int(202),
				Minor:   int(1),
				PCIPath: "pcipath-xxx",
			},
			expectedResult: true,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sda",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
					PCIPath:     "pcipath-xxx",
				},
			},
			device: &sys.Device{
				Name:    "sda",
				Major:   int(202),
				Minor:   int(1),
				PCIPath: "pcipath-yyy",
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sda",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
				},
			},
			device: &sys.Device{
				Name:  "sda",
				Major: int(202),
				Minor: int(1),
			},
			expectedResult: true,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sdb",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
				},
			},
			device: &sys.Device{
				Name:  "sda",
				Major: int(202),
				Minor: int(1),
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sda",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
				},
			},
			device: &sys.Device{
				Name:  "sda",
				Major: int(203),
				Minor: int(1),
			},
			expectedResult: false,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:        "/dev/sda",
					MajorNumber: uint32(202),
					MinorNumber: uint32(1),
				},
			},
			device: &sys.Device{
				Name:  "sda",
				Major: int(202),
				Minor: int(2),
			},
			expectedResult: false,
		},
	}
	for i, testCase := range testCases {
		result := nonConclusiveMatcher(testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Errorf("case %d expected result: %v but got %v", i, testCase.expectedResult, result)
		}
	}
}

func TestMatchDrives(t *testing.T) {
	testCases := []struct {
		drives              []*directcsi.DirectCSIDrive
		device              *sys.Device
		expectedDrive       *directcsi.DirectCSIDrive
		expectedMatchResult matchResult
	}{
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "lasbfqkp-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedDrive:       nil,
			expectedMatchResult: noMatch,
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
						Path:           "/dev/sda",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-2",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				Name:   "sda",
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Path:           "/dev/sda",
				},
			},
			expectedMatchResult: noChange,
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Spec: directcsi.DirectCSIDriveSpec{
						RequestedFormat: &directcsi.RequestedFormat{
							Force:      true,
							Filesystem: "xfs",
						},
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-2",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Spec: directcsi.DirectCSIDriveSpec{
					RequestedFormat: &directcsi.RequestedFormat{
						Force:      true,
						Filesystem: "xfs",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				},
			},
			expectedMatchResult: changed,
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
						Path:           "/dev/sdb",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "z5db531k-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-2",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
				Name:   "sda",
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-drive",
					Namespace: metav1.NamespaceNone,
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					Path:           "/dev/sdb",
				},
			},
			expectedMatchResult: changed,
		},
		{
			drives: []*directcsi.DirectCSIDrive{
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
				{
					TypeMeta: utils.DirectCSIDriveTypeMeta(),
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-drive-1",
						Namespace: metav1.NamespaceNone,
					},
					Status: directcsi.DirectCSIDriveStatus{
						FilesystemUUID: "pgjwblpw-0d9d-4e6e-a766-c79ac18b7ea6",
					},
				},
			},
			device: &sys.Device{
				FSUUID: "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
			},
			expectedDrive:       nil,
			expectedMatchResult: tooManyMatches,
		},
	}

	for i, testCase := range testCases {
		drive, matchResult := matchDrives(testCase.drives, testCase.device)
		if !reflect.DeepEqual(drive, testCase.expectedDrive) {
			t.Errorf("case %d expected drive: %v but got %v", i, testCase.expectedDrive, drive)
		}
		if matchResult != testCase.expectedMatchResult {
			t.Errorf("case %d expected matchResult: %v but got %v", i, testCase.expectedMatchResult, matchResult)
		}
	}
}
