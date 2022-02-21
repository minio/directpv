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

	"github.com/minio/directpv/pkg/sys"
)

func TestMapToEventData(t *testing.T) {
	testEventMap := map[string]string{
		"DEVPATH":              "/devices/virtual/block/loop7",
		"MAJOR":                "7",
		"MINOR":                "0",
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

	expectedUEventData := &sys.UDevData{
		Path:         "/devices/virtual/block/loop7",
		Major:        7,
		Minor:        0,
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

	udevData, err := mapToUdevData(testEventMap)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !reflect.DeepEqual(udevData, expectedUEventData) {
		t.Fatalf("expected udevdata: %v, got: %v", udevData, expectedUEventData)
	}
}
