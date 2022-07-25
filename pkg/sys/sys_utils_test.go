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

package sys

import (
	"reflect"
	"testing"
)

func TestMapToEventData(t *testing.T) {
	testEventMap := map[string]string{
		"MD_UUID":              "MDUUID",
		"ID_PART_ENTRY_NUMBER": "7",
		"ID_WWN":               "WWN",
		"ID_MODEL":             "ID_MODEL",
		"ID_SERIAL_SHORT":      "ID_SERIAL_SHORT",
		"ID_VENDOR":            "ID_VENDOR",
		"DM_NAME":              "DM_NAME",
		"DM_UUID":              "DM_UUID",
		"ID_PART_TABLE_UUID":   "ID_PART_TABLE_UUID",
		"ID_PART_TABLE_TYPE":   "ID_PART_TABLE_TYPE",
		"ID_PART_ENTRY_UUID":   "ID_PART_ENTRY_UUID",
		"ID_FS_UUID":           "ID_FS_UUID",
		"ID_FS_TYPE":           "ID_FS_TYPE",
	}

	expectedUEventData := &UDevData{
		Partition:    7,
		WWID:         "WWN",
		Model:        "ID_MODEL",
		UeventSerial: "ID_SERIAL_SHORT",
		Vendor:       "ID_VENDOR",
		DMName:       "DM_NAME",
		DMUUID:       "DM_UUID",
		MDUUID:       "MDUUID",
		PTUUID:       "ID_PART_TABLE_UUID",
		PTType:       "ID_PART_TABLE_TYPE",
		PartUUID:     "ID_PART_ENTRY_UUID",
		UeventFSUUID: "ID_FS_UUID",
		FSType:       "ID_FS_TYPE",
	}

	udevData, err := MapToUdevData(testEventMap)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(udevData, expectedUEventData) {
		t.Fatalf("expected udevdata: %v, got: %v", udevData, expectedUEventData)
	}
}

func TestIsLVMMemberFSType(t *testing.T) {
	testCases := []struct {
		fsType         string
		expectedResult bool
	}{
		{"LVM2_member", true},
		{"lvm2_member", true},
		{"xfs", false},
	}

	for i, testCase := range testCases {
		result := isLVMMemberFSType(testCase.fsType)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsDeviceUnavailable(t *testing.T) {
	testCases := []struct {
		device         *Device
		expectedResult bool
	}{
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: false,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize - 1, // drive with size less then supported
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          true, // swapons
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          true, // hidden device
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        true, // readonly device
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           true, // cdrom device
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     true, // partitioned device
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "/dev/sda", // master
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{"/dev/dm-0"}, // has holders
				FirstMountPoint: "",
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "/mnt/abc", // mounted drive
				FSType:          "xfs",
			},
			expectedResult: true,
		},
		{
			device: &Device{
				Size:            MinSupportedDeviceSize,
				SwapOn:          false,
				Hidden:          false,
				ReadOnly:        false,
				Removable:       false,
				CDRom:           false,
				Partitioned:     false,
				Master:          "",
				Holders:         []string{},
				FirstMountPoint: "",
				FSType:          "LVM2_member", // lvm configured drive
			},
			expectedResult: true,
		},
	}

	for i, testCase := range testCases {
		result := IsDeviceUnavailable(testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}
