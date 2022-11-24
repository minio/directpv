// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package device

import (
	"testing"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
)

func TestID(t *testing.T) {
	testCases := []struct {
		device     Device
		expectedID string
	}{
		{
			device: Device{
				Name:        "nvne1n1",
				MajorMinor:  "259:0",
				Size:        401293127,
				Hidden:      false,
				Removable:   false,
				ReadOnly:    false,
				Partitioned: false,
				Holders:     nil,
				MountPoints: []string{"/mnt/a", "/mnt/b", "/mnt/c"},
				SwapOn:      false,
				CDROM:       false,
				DMName:      "",
				UDevData: map[string]string{
					"ID_SERIAL_SHORT":      "FBFB18060MY0001903",
					"ID_WWN":               "eui.a03299af790f1001",
					"ID_MODEL":             "LENSE20256GMSP34MEAT2TA",
					"ID_REVISION":          "2.8.8341",
					"ID_SERIAL":            "LENSE20256GMSP34MEAT2TA_FBFB18060MY0001903",
					"ID_PATH":              "pci-0000:04:00.0-nvme-1",
					"ID_PATH_TAG":          "pci-0000_04_00_0-nvme-1",
					"ID_PART_TABLE_TYPE":   "gpt",
					"ID_FS_UUID":           "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_UUID_ENC":       "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_VERSION":        "1.0",
					"ID_FS_TYPE":           "ext4",
					"ID_FS_USAGE":          "filesystem",
					"ID_PART_ENTRY_SCHEME": "gpt",
					"ID_PART_ENTRY_UUID":   "6fa7b66d-4e3b-4c1d-b9b3-2fc5779f93a0",
					"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
					"ID_PART_ENTRY_NUMBER": "2",
					"ID_PART_ENTRY_OFFSET": "1050624",
					"ID_PART_ENTRY_SIZE":   "401293127",
					"ID_PART_ENTRY_DISK":   "259",
				},
			},
			expectedID: "aYcXwG7DITo7eOPrSCAT9rTa6CK3hjxpC21rJ1v7Kqk=",
		},
		// with mountpoints disordered
		{
			device: Device{
				Name:        "nvne1n1",
				MajorMinor:  "259:0",
				Size:        401293127,
				Hidden:      false,
				Removable:   false,
				ReadOnly:    false,
				Partitioned: false,
				Holders:     nil,
				MountPoints: []string{"/mnt/c", "/mnt/b", "/mnt/a"},
				SwapOn:      false,
				CDROM:       false,
				DMName:      "",
				UDevData: map[string]string{
					"ID_SERIAL_SHORT":      "FBFB18060MY0001903",
					"ID_WWN":               "eui.a03299af790f1001",
					"ID_MODEL":             "LENSE20256GMSP34MEAT2TA",
					"ID_REVISION":          "2.8.8341",
					"ID_SERIAL":            "LENSE20256GMSP34MEAT2TA_FBFB18060MY0001903",
					"ID_PATH":              "pci-0000:04:00.0-nvme-1",
					"ID_PATH_TAG":          "pci-0000_04_00_0-nvme-1",
					"ID_PART_TABLE_TYPE":   "gpt",
					"ID_FS_UUID":           "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_UUID_ENC":       "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_VERSION":        "1.0",
					"ID_FS_TYPE":           "ext4",
					"ID_FS_USAGE":          "filesystem",
					"ID_PART_ENTRY_SCHEME": "gpt",
					"ID_PART_ENTRY_UUID":   "6fa7b66d-4e3b-4c1d-b9b3-2fc5779f93a0",
					"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
					"ID_PART_ENTRY_NUMBER": "2",
					"ID_PART_ENTRY_OFFSET": "1050624",
					"ID_PART_ENTRY_SIZE":   "401293127",
					"ID_PART_ENTRY_DISK":   "259",
				},
			},
			expectedID: "aYcXwG7DITo7eOPrSCAT9rTa6CK3hjxpC21rJ1v7Kqk=",
		},
		// with udevdata disordered
		{
			device: Device{
				Name:        "nvne1n1",
				MajorMinor:  "259:0",
				Size:        401293127,
				Hidden:      false,
				Removable:   false,
				ReadOnly:    false,
				Partitioned: false,
				Holders:     nil,
				MountPoints: []string{"/mnt/c", "/mnt/b", "/mnt/a"},
				SwapOn:      false,
				CDROM:       false,
				DMName:      "",
				UDevData: map[string]string{
					"ID_SERIAL_SHORT":      "FBFB18060MY0001903",
					"ID_WWN":               "eui.a03299af790f1001",
					"ID_SERIAL":            "LENSE20256GMSP34MEAT2TA_FBFB18060MY0001903",
					"ID_PATH":              "pci-0000:04:00.0-nvme-1",
					"ID_PATH_TAG":          "pci-0000_04_00_0-nvme-1",
					"ID_PART_TABLE_TYPE":   "gpt",
					"ID_FS_USAGE":          "filesystem",
					"ID_PART_ENTRY_SCHEME": "gpt",
					"ID_PART_ENTRY_UUID":   "6fa7b66d-4e3b-4c1d-b9b3-2fc5779f93a0",
					"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
					"ID_PART_ENTRY_NUMBER": "2",
					"ID_PART_ENTRY_OFFSET": "1050624",
					"ID_PART_ENTRY_SIZE":   "401293127",
					"ID_PART_ENTRY_DISK":   "259",
					"ID_FS_UUID":           "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_UUID_ENC":       "a5eb531b-0d9d-4e6e-a766-c79ac18b7ea6",
					"ID_FS_VERSION":        "1.0",
					"ID_FS_TYPE":           "ext4",
					"ID_MODEL":             "LENSE20256GMSP34MEAT2TA",
					"ID_REVISION":          "2.8.8341",
				},
			},
			expectedID: "aYcXwG7DITo7eOPrSCAT9rTa6CK3hjxpC21rJ1v7Kqk=",
		},
	}

	for i, testCase := range testCases {
		generatedID := testCase.device.ID(types.NodeID("node-1"))
		if testCase.expectedID != generatedID {
			t.Fatalf("case %v: expected: %+v; got: %+v", i+1, testCase.expectedID, generatedID)
		}
	}
}
