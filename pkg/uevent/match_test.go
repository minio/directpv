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
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRunMatchers(t *testing.T) {
	validDriveObjs := []*directcsi.DirectCSIDrive{
		{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_1",
			},
			Status: directcsi.DirectCSIDriveStatus{
				UeventSerial:   "SERIAL1",
				FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
			},
		},
		{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_2",
			},
			Status: directcsi.DirectCSIDriveStatus{
				UeventSerial:   "SERIAL2",
				FilesystemUUID: "ertsdfff-e1b5-4bac-b73f-178b29974ed5",
			},
		},
	}

	terminatingDriveObjects := []*directcsi.DirectCSIDrive{
		{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_3",
			},
			Status: directcsi.DirectCSIDriveStatus{
				UeventSerial:   "SERIAL2",
				FilesystemUUID: "ertsdfff-e1b5-4bac-b73f-178b29974ed5",
				DriveStatus:    directcsi.DriveStatusTerminating,
			},
		},
		{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_drive_4",
			},
			Status: directcsi.DirectCSIDriveStatus{
				UeventSerial:   "SERIAL3",
				FilesystemUUID: "ertsdfff-e1b5-4bac-b73f-178b29974ed5",
				DriveStatus:    directcsi.DriveStatusTerminating,
			},
		},
	}

	driveObjects := append(validDriveObjs, terminatingDriveObjects...)

	testDevice := &sys.Device{
		FSUUID:       "d9877501-e1b5-4bac-b73f-178b29974ed5",
		UeventSerial: "SERIAL1",
	}

	var matchCounter int
	var stageTwoHit bool

	testCases := []struct {
		name                string
		stageOnematchers    []matchFn
		stageTwoMatchers    []matchFn
		stageTwoHit         bool
		expectedDrive       *directcsi.DirectCSIDrive
		expectedMatchResult matchResult
		expectedMatchHit    int
	}{
		{
			name: "no_match_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = false
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = false
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					return
				},
			},
			stageTwoHit:         false,
			expectedDrive:       nil,
			expectedMatchResult: noMatch,
			expectedMatchHit:    1 * len(validDriveObjs),
		},
		{
			name: "more_than_one_considered_drives_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = true
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = true
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					match = true
					consider = false
					return
				},
			},
			stageTwoHit:         true,
			expectedDrive:       nil,
			expectedMatchResult: tooManyMatches,
			expectedMatchHit:    2 * len(validDriveObjs),
		},
		{
			name: "one_considered_drive_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = drive.Status.UeventSerial == device.UeventSerial
					consider = false
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = true
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = false
					consider = true
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					return
				},
			},
			stageTwoHit: false,
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_drive_1",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
					UeventSerial:   "SERIAL1",
				},
			},
			expectedMatchResult: changed,
			expectedMatchHit:    1*len(validDriveObjs) + 1 + 1,
		},
		{
			name: "more_than_one_matched_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = true
					consider = false
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = true
					consider = false
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					match = true
					consider = true
					return
				},
			},
			stageTwoHit:         true,
			expectedDrive:       nil,
			expectedMatchResult: tooManyMatches,
			expectedMatchHit:    2 * len(validDriveObjs),
		},
		{
			name: "one_matched_drive_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = drive.Status.UeventSerial == device.UeventSerial
					consider = false
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = drive.Status.FilesystemUUID == device.FSUUID
					consider = false
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					return
				},
			},
			stageTwoHit: false,
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_drive_1",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
					UeventSerial:   "SERIAL1",
				},
			},
			expectedMatchResult: noChange,
			expectedMatchHit:    1*len(validDriveObjs) + 1,
		},
		{
			name: "matched_and_considered_drives_test",
			stageOnematchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = true
					consider = true
					return
				},
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					matchCounter++
					match = drive.Status.FilesystemUUID == device.FSUUID
					consider = true
					return
				},
			},
			stageTwoMatchers: []matchFn{
				func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
					stageTwoHit = true
					return
				},
			},
			stageTwoHit: false,
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_drive_1",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "d9877501-e1b5-4bac-b73f-178b29974ed5",
					UeventSerial:   "SERIAL1",
				},
			},
			expectedMatchResult: noChange,
			expectedMatchHit:    2 * len(validDriveObjs),
		},
	}

	for _, tt := range testCases {
		matchCounter = 0
		stageTwoHit = false
		drive, matchResult := runMatchers(driveObjects, testDevice, tt.stageOnematchers, tt.stageTwoMatchers)
		if !reflect.DeepEqual(drive, tt.expectedDrive) {
			t.Errorf("test: %s expected drive: %v but got: %v", tt.name, tt.expectedDrive, drive)
		}
		if matchCounter != tt.expectedMatchHit {
			t.Errorf("test: %s expected mactchHit: %d but got: %d", tt.name, tt.expectedMatchHit, matchCounter)
		}
		if matchResult != tt.expectedMatchResult {
			t.Errorf("test: %s expected matchResult: %v but got: %v", tt.name, tt.expectedMatchResult, matchResult)
		}
		if stageTwoHit != tt.stageTwoHit {
			t.Errorf("test: %s expected stageTwoHit: %v but got: %v", tt.name, tt.stageTwoHit, stageTwoHit)
		}
	}
}
