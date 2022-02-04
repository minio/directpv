//go:build linux

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
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func getCase1DataResult() (string, map[string]string) {
	data := `I:8854899
E:ID_FS_TYPE=
G:systemd
`
	result := map[string]string{"ID_FS_TYPE": ""}
	return data, result
}

func getCase2DataResult() (string, map[string]string) {
	data := `S:disk/by-path/pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0
S:disk/by-id/usb-SanDisk_Ultra_4C531234567891234567-0:0
W:9
I:50939541932
E:ID_VENDOR=SanDisk
E:ID_VENDOR_ENC=SanDisk\x20
E:ID_VENDOR_ID=0781
E:ID_MODEL=Ultra
E:ID_MODEL_ENC=Ultra\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20
E:ID_MODEL_ID=558a
E:ID_REVISION=1.00
E:ID_SERIAL=SanDisk_Ultra_4C531234567891234567-0:0
E:ID_SERIAL_SHORT=4C531234567891234567
E:ID_TYPE=disk
E:ID_INSTANCE=0:0
E:ID_BUS=usb
E:ID_USB_INTERFACES=:080650:
E:ID_USB_INTERFACE_NUM=00
E:ID_USB_DRIVER=usb-storage
E:ID_PATH=pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0
E:ID_PATH_TAG=pci-0000_00_14_0-usb-0_4_1_0-scsi-0_0_0_0
E:ID_PART_TABLE_TYPE=dos
E:ID_FS_TYPE=
G:systemd
`
	result := map[string]string{
		"ID_VENDOR":            "SanDisk",
		"ID_VENDOR_ENC":        "SanDisk\\x20",
		"ID_VENDOR_ID":         "0781",
		"ID_MODEL":             "Ultra",
		"ID_MODEL_ENC":         "Ultra\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20",
		"ID_MODEL_ID":          "558a",
		"ID_REVISION":          "1.00",
		"ID_SERIAL":            "SanDisk_Ultra_4C531234567891234567-0:0",
		"ID_SERIAL_SHORT":      "4C531234567891234567",
		"ID_TYPE":              "disk",
		"ID_INSTANCE":          "0:0",
		"ID_BUS":               "usb",
		"ID_USB_INTERFACES":    ":080650:",
		"ID_USB_INTERFACE_NUM": "00",
		"ID_USB_DRIVER":        "usb-storage",
		"ID_PATH":              "pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0",
		"ID_PATH_TAG":          "pci-0000_00_14_0-usb-0_4_1_0-scsi-0_0_0_0",
		"ID_PART_TABLE_TYPE":   "dos",
		"ID_FS_TYPE":           "",
	}
	return data, result
}

func getCase3DataResult() (string, map[string]string) {
	data := `S:disk/by-path/pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0-part1
S:disk/by-uuid/1234-ABCD
S:disk/by-id/usb-SanDisk_Ultra_4C531234567891234567-0:0-part1
W:11
I:50939590330
E:ID_VENDOR=SanDisk
E:ID_VENDOR_ENC=SanDisk\x20
E:ID_VENDOR_ID=0781
E:ID_MODEL=Ultra
E:ID_MODEL_ENC=Ultra\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20\x20
E:ID_MODEL_ID=558a
E:ID_REVISION=1.00
E:ID_SERIAL=SanDisk_Ultra_4C531234567891234567-0:0
E:ID_SERIAL_SHORT=4C531234567891234567
E:ID_TYPE=disk
E:ID_INSTANCE=0:0
E:ID_BUS=usb
E:ID_USB_INTERFACES=:080650:
E:ID_USB_INTERFACE_NUM=00
E:ID_USB_DRIVER=usb-storage
E:ID_PATH=pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0
E:ID_PATH_TAG=pci-0000_00_14_0-usb-0_4_1_0-scsi-0_0_0_0
E:ID_PART_TABLE_TYPE=dos
E:ID_FS_UUID=1234-ABCD
E:ID_FS_UUID_ENC=1234-ABCD
E:ID_FS_VERSION=FAT32
E:ID_FS_TYPE=vfat
E:ID_FS_USAGE=filesystem
E:ID_PART_ENTRY_SCHEME=dos
E:ID_PART_ENTRY_TYPE=0xc
E:ID_PART_ENTRY_NUMBER=1
E:ID_PART_ENTRY_OFFSET=32
E:ID_PART_ENTRY_SIZE=121307104
E:ID_PART_ENTRY_DISK=8:0
G:systemd
`
	result := map[string]string{
		"ID_VENDOR":            "SanDisk",
		"ID_VENDOR_ENC":        "SanDisk\\x20",
		"ID_VENDOR_ID":         "0781",
		"ID_MODEL":             "Ultra",
		"ID_MODEL_ENC":         "Ultra\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20\\x20",
		"ID_MODEL_ID":          "558a",
		"ID_REVISION":          "1.00",
		"ID_SERIAL":            "SanDisk_Ultra_4C531234567891234567-0:0",
		"ID_SERIAL_SHORT":      "4C531234567891234567",
		"ID_TYPE":              "disk",
		"ID_INSTANCE":          "0:0",
		"ID_BUS":               "usb",
		"ID_USB_INTERFACES":    ":080650:",
		"ID_USB_INTERFACE_NUM": "00",
		"ID_USB_DRIVER":        "usb-storage",
		"ID_PATH":              "pci-0000:00:14.0-usb-0:4:1.0-scsi-0:0:0:0",
		"ID_PATH_TAG":          "pci-0000_00_14_0-usb-0_4_1_0-scsi-0_0_0_0",
		"ID_PART_TABLE_TYPE":   "dos",
		"ID_FS_UUID":           "1234-ABCD",
		"ID_FS_UUID_ENC":       "1234-ABCD",
		"ID_FS_VERSION":        "FAT32",
		"ID_FS_TYPE":           "vfat",
		"ID_FS_USAGE":          "filesystem",
		"ID_PART_ENTRY_SCHEME": "dos",
		"ID_PART_ENTRY_TYPE":   "0xc",
		"ID_PART_ENTRY_NUMBER": "1",
		"ID_PART_ENTRY_OFFSET": "32",
		"ID_PART_ENTRY_SIZE":   "121307104",
		"ID_PART_ENTRY_DISK":   "8:0",
	}
	return data, result
}

func getCase4DataResult() (string, map[string]string) {
	data := `S:disk/by-id/nvme-Micron_2300_NVMe_512GB__12345ABCD678
S:disk/by-id/nvme-eui.000000000000000200a012304bd678b5
S:disk/by-path/pci-0000:6d:00.0-nvme-1
W:1
I:8399206
E:ID_SERIAL_SHORT=        12345ABCD678
E:ID_WWN=eui.000000000000000200a012304bd678b5
E:ID_MODEL=Micron 2300 NVMe 512GB
E:ID_REVISION=23000020
E:ID_SERIAL=Micron 2300 NVMe 512GB_        12345ABCD678
E:ID_PATH=pci-0000:6d:00.0-nvme-1
E:ID_PATH_TAG=pci-0000_6d_00_0-nvme-1
E:ID_PART_TABLE_UUID=27c9e87c-45b2-44eb-b0be-cf52b7d47794
E:ID_PART_TABLE_TYPE=gpt
E:ID_FS_TYPE=
G:systemd
`
	result := map[string]string{
		"ID_SERIAL_SHORT":    "12345ABCD678",
		"ID_WWN":             "eui.000000000000000200a012304bd678b5",
		"ID_MODEL":           "Micron 2300 NVMe 512GB",
		"ID_REVISION":        "23000020",
		"ID_SERIAL":          "Micron 2300 NVMe 512GB_        12345ABCD678",
		"ID_PATH":            "pci-0000:6d:00.0-nvme-1",
		"ID_PATH_TAG":        "pci-0000_6d_00_0-nvme-1",
		"ID_PART_TABLE_UUID": "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		"ID_PART_TABLE_TYPE": "gpt",
		"ID_FS_TYPE":         "",
	}
	return data, result
}

func getCase5DataResult() (string, map[string]string) {
	data := `S:disk/by-partuuid/9a69a545-28c3-441c-a60b-6ed5223b03c3
S:disk/by-id/nvme-eui.000000000000000200a012304bd678b5-part1
S:disk/by-path/pci-0000:6d:00.0-nvme-1-part1
S:disk/by-id/nvme-Micron_2300_NVMe_512GB__12345ABCD678-part1
S:disk/by-partlabel/EFI\x20System\x20Partition
S:disk/by-uuid/4321-FEDC
W:8
I:8405991
E:ID_SERIAL_SHORT=        12345ABCD678
E:ID_WWN=eui.000000000000000200a012304bd678b5
E:ID_MODEL=Micron 2300 NVMe 512GB
E:ID_REVISION=23000020
E:ID_SERIAL=Micron 2300 NVMe 512GB_        12345ABCD678
E:ID_PATH=pci-0000:6d:00.0-nvme-1
E:ID_PATH_TAG=pci-0000_6d_00_0-nvme-1
E:ID_PART_TABLE_UUID=27c9e87c-45b2-44eb-b0be-cf52b7d47794
E:ID_PART_TABLE_TYPE=gpt
E:ID_FS_UUID=4321-FEDC
E:ID_FS_UUID_ENC=4321-FEDC
E:ID_FS_VERSION=FAT32
E:ID_FS_TYPE=vfat
E:ID_FS_USAGE=filesystem
E:ID_PART_ENTRY_SCHEME=gpt
E:ID_PART_ENTRY_NAME=EFI\x20System\x20Partition
E:ID_PART_ENTRY_UUID=9a69a545-28c3-441c-a60b-6ed5223b03c3
E:ID_PART_ENTRY_TYPE=c12a7328-f81f-11d2-ba4b-00a0c93ec93b
E:ID_PART_ENTRY_NUMBER=1
E:ID_PART_ENTRY_OFFSET=2048
E:ID_PART_ENTRY_SIZE=1228800
E:ID_PART_ENTRY_DISK=259:0
E:UDISKS_IGNORE=1
G:systemd
`
	result := map[string]string{
		"ID_SERIAL_SHORT":      "12345ABCD678",
		"ID_WWN":               "eui.000000000000000200a012304bd678b5",
		"ID_MODEL":             "Micron 2300 NVMe 512GB",
		"ID_REVISION":          "23000020",
		"ID_SERIAL":            "Micron 2300 NVMe 512GB_        12345ABCD678",
		"ID_PATH":              "pci-0000:6d:00.0-nvme-1",
		"ID_PATH_TAG":          "pci-0000_6d_00_0-nvme-1",
		"ID_PART_TABLE_UUID":   "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		"ID_PART_TABLE_TYPE":   "gpt",
		"ID_FS_UUID":           "4321-FEDC",
		"ID_FS_UUID_ENC":       "4321-FEDC",
		"ID_FS_VERSION":        "FAT32",
		"ID_FS_TYPE":           "vfat",
		"ID_FS_USAGE":          "filesystem",
		"ID_PART_ENTRY_SCHEME": "gpt",
		"ID_PART_ENTRY_NAME":   "EFI\\x20System\\x20Partition",
		"ID_PART_ENTRY_UUID":   "9a69a545-28c3-441c-a60b-6ed5223b03c3",
		"ID_PART_ENTRY_TYPE":   "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
		"ID_PART_ENTRY_NUMBER": "1",
		"ID_PART_ENTRY_OFFSET": "2048",
		"ID_PART_ENTRY_SIZE":   "1228800",
		"ID_PART_ENTRY_DISK":   "259:0",
		"UDISKS_IGNORE":        "1",
	}
	return data, result
}

func getCase6DataResult() (string, map[string]string) {
	data := `S:disk/by-path/pci-0000:6d:00.0-nvme-1-part2
S:disk/by-partuuid/0959536f-134a-477f-9c02-a7916d034a33
S:disk/by-uuid/9b7c849b-387e-43f8-ad5a-b7d68c5c062f
S:disk/by-id/nvme-Micron_2300_NVMe_512GB__12345ABCD678-part2
S:disk/by-id/nvme-eui.000000000000000200a012304bd678b5-part2
W:6
I:8405220
E:ID_SERIAL_SHORT=        12345ABCD678
E:ID_WWN=eui.000000000000000200a012304bd678b5
E:ID_MODEL=Micron 2300 NVMe 512GB
E:ID_REVISION=23000020
E:ID_SERIAL=Micron 2300 NVMe 512GB_        12345ABCD678
E:ID_PATH=pci-0000:6d:00.0-nvme-1
E:ID_PATH_TAG=pci-0000_6d_00_0-nvme-1
E:ID_PART_TABLE_UUID=27c9e87c-45b2-44eb-b0be-cf52b7d47794
E:ID_PART_TABLE_TYPE=gpt
E:ID_FS_UUID=9b7c849b-387e-43f8-ad5a-b7d68c5c062f
E:ID_FS_UUID_ENC=9b7c849b-387e-43f8-ad5a-b7d68c5c062f
E:ID_FS_VERSION=1.0
E:ID_FS_TYPE=ext4
E:ID_FS_USAGE=filesystem
E:ID_PART_ENTRY_SCHEME=gpt
E:ID_PART_ENTRY_UUID=0959536f-134a-477f-9c02-a7916d034a33
E:ID_PART_ENTRY_TYPE=0fc63daf-8483-4772-8e79-3d69d8477de4
E:ID_PART_ENTRY_NUMBER=2
E:ID_PART_ENTRY_OFFSET=1230848
E:ID_PART_ENTRY_SIZE=2097152
E:ID_PART_ENTRY_DISK=259:0
G:systemd
`
	result := map[string]string{
		"ID_SERIAL_SHORT":      "12345ABCD678",
		"ID_WWN":               "eui.000000000000000200a012304bd678b5",
		"ID_MODEL":             "Micron 2300 NVMe 512GB",
		"ID_REVISION":          "23000020",
		"ID_SERIAL":            "Micron 2300 NVMe 512GB_        12345ABCD678",
		"ID_PATH":              "pci-0000:6d:00.0-nvme-1",
		"ID_PATH_TAG":          "pci-0000_6d_00_0-nvme-1",
		"ID_PART_TABLE_UUID":   "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		"ID_PART_TABLE_TYPE":   "gpt",
		"ID_FS_UUID":           "9b7c849b-387e-43f8-ad5a-b7d68c5c062f",
		"ID_FS_UUID_ENC":       "9b7c849b-387e-43f8-ad5a-b7d68c5c062f",
		"ID_FS_VERSION":        "1.0",
		"ID_FS_TYPE":           "ext4",
		"ID_FS_USAGE":          "filesystem",
		"ID_PART_ENTRY_SCHEME": "gpt",
		"ID_PART_ENTRY_UUID":   "0959536f-134a-477f-9c02-a7916d034a33",
		"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
		"ID_PART_ENTRY_NUMBER": "2",
		"ID_PART_ENTRY_OFFSET": "1230848",
		"ID_PART_ENTRY_SIZE":   "2097152",
		"ID_PART_ENTRY_DISK":   "259:0",
	}
	return data, result
}

func getCase7DataResult() (string, map[string]string) {
	data := `S:disk/by-path/pci-0000:6d:00.0-nvme-1-part3
S:disk/by-id/nvme-eui.000000000000000200a012304bd678b5-part3
S:disk/by-uuid/1c9fee93-cc76-4d9d-a1b1-9895c06df6e3
S:disk/by-partuuid/f8b65530-58bb-4464-b257-d3cb2aba034b
S:disk/by-id/nvme-Micron_2300_NVMe_512GB__12345ABCD678-part3
W:5
I:8411345
E:ID_SERIAL_SHORT=        12345ABCD678
E:ID_WWN=eui.000000000000000200a012304bd678b5
E:ID_MODEL=Micron 2300 NVMe 512GB
E:ID_REVISION=23000020
E:ID_SERIAL=Micron 2300 NVMe 512GB_        12345ABCD678
E:ID_PATH=pci-0000:6d:00.0-nvme-1
E:ID_PATH_TAG=pci-0000_6d_00_0-nvme-1
E:ID_PART_TABLE_UUID=27c9e87c-45b2-44eb-b0be-cf52b7d47794
E:ID_PART_TABLE_TYPE=gpt
E:ID_FS_UUID=1c9fee93-cc76-4d9d-a1b1-9895c06df6e3
E:ID_FS_UUID_ENC=1c9fee93-cc76-4d9d-a1b1-9895c06df6e3
E:ID_FS_VERSION=1.0
E:ID_FS_TYPE=ext4
E:ID_FS_USAGE=filesystem
E:ID_PART_ENTRY_SCHEME=gpt
E:ID_PART_ENTRY_UUID=f8b65530-58bb-4464-b257-d3cb2aba034b
E:ID_PART_ENTRY_TYPE=0fc63daf-8483-4772-8e79-3d69d8477de4
E:ID_PART_ENTRY_NUMBER=3
E:ID_PART_ENTRY_OFFSET=3328000
E:ID_PART_ENTRY_SIZE=104857600
E:ID_PART_ENTRY_DISK=259:0
G:systemd
`
	result := map[string]string{
		"ID_SERIAL_SHORT":      "12345ABCD678",
		"ID_WWN":               "eui.000000000000000200a012304bd678b5",
		"ID_MODEL":             "Micron 2300 NVMe 512GB",
		"ID_REVISION":          "23000020",
		"ID_SERIAL":            "Micron 2300 NVMe 512GB_        12345ABCD678",
		"ID_PATH":              "pci-0000:6d:00.0-nvme-1",
		"ID_PATH_TAG":          "pci-0000_6d_00_0-nvme-1",
		"ID_PART_TABLE_UUID":   "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		"ID_PART_TABLE_TYPE":   "gpt",
		"ID_FS_UUID":           "1c9fee93-cc76-4d9d-a1b1-9895c06df6e3",
		"ID_FS_UUID_ENC":       "1c9fee93-cc76-4d9d-a1b1-9895c06df6e3",
		"ID_FS_VERSION":        "1.0",
		"ID_FS_TYPE":           "ext4",
		"ID_FS_USAGE":          "filesystem",
		"ID_PART_ENTRY_SCHEME": "gpt",
		"ID_PART_ENTRY_UUID":   "f8b65530-58bb-4464-b257-d3cb2aba034b",
		"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
		"ID_PART_ENTRY_NUMBER": "3",
		"ID_PART_ENTRY_OFFSET": "3328000",
		"ID_PART_ENTRY_SIZE":   "104857600",
		"ID_PART_ENTRY_DISK":   "259:0",
	}
	return data, result
}

func getCase8DataResult() (string, map[string]string) {
	data := `S:disk/by-partuuid/656b4db9-e2f9-42a6-b09e-9466dcb070dc
S:disk/by-path/pci-0000:6d:00.0-nvme-1-part4
S:disk/by-uuid/a49f8e69-03fb-4735-b900-91d068fcbb70
S:disk/by-id/nvme-eui.000000000000000200a012304bd678b5-part4
S:disk/by-id/nvme-Micron_2300_NVMe_512GB__12345ABCD678-part4
W:7
I:8411038
E:ID_SERIAL_SHORT=        12345ABCD678
E:ID_WWN=eui.000000000000000200a012304bd678b5
E:ID_MODEL=Micron 2300 NVMe 512GB
E:ID_REVISION=23000020
E:ID_SERIAL=Micron 2300 NVMe 512GB_        12345ABCD678
E:ID_PATH=pci-0000:6d:00.0-nvme-1
E:ID_PATH_TAG=pci-0000_6d_00_0-nvme-1
E:ID_PART_TABLE_UUID=27c9e87c-45b2-44eb-b0be-cf52b7d47794
E:ID_PART_TABLE_TYPE=gpt
E:ID_FS_UUID=a49f8e69-03fb-4735-b900-91d068fcbb70
E:ID_FS_UUID_ENC=a49f8e69-03fb-4735-b900-91d068fcbb70
E:ID_FS_VERSION=1.0
E:ID_FS_TYPE=ext4
E:ID_FS_USAGE=filesystem
E:ID_PART_ENTRY_SCHEME=gpt
E:ID_PART_ENTRY_UUID=656b4db9-e2f9-42a6-b09e-9466dcb070dc
E:ID_PART_ENTRY_TYPE=0fc63daf-8483-4772-8e79-3d69d8477de4
E:ID_PART_ENTRY_NUMBER=4
E:ID_PART_ENTRY_OFFSET=108185600
E:ID_PART_ENTRY_SIZE=892028928
E:ID_PART_ENTRY_DISK=259:0
G:systemd
`
	result := map[string]string{
		"ID_SERIAL_SHORT":      "12345ABCD678",
		"ID_WWN":               "eui.000000000000000200a012304bd678b5",
		"ID_MODEL":             "Micron 2300 NVMe 512GB",
		"ID_REVISION":          "23000020",
		"ID_SERIAL":            "Micron 2300 NVMe 512GB_        12345ABCD678",
		"ID_PATH":              "pci-0000:6d:00.0-nvme-1",
		"ID_PATH_TAG":          "pci-0000_6d_00_0-nvme-1",
		"ID_PART_TABLE_UUID":   "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		"ID_PART_TABLE_TYPE":   "gpt",
		"ID_FS_UUID":           "a49f8e69-03fb-4735-b900-91d068fcbb70",
		"ID_FS_UUID_ENC":       "a49f8e69-03fb-4735-b900-91d068fcbb70",
		"ID_FS_VERSION":        "1.0",
		"ID_FS_TYPE":           "ext4",
		"ID_FS_USAGE":          "filesystem",
		"ID_PART_ENTRY_SCHEME": "gpt",
		"ID_PART_ENTRY_UUID":   "656b4db9-e2f9-42a6-b09e-9466dcb070dc",
		"ID_PART_ENTRY_TYPE":   "0fc63daf-8483-4772-8e79-3d69d8477de4",
		"ID_PART_ENTRY_NUMBER": "4",
		"ID_PART_ENTRY_OFFSET": "108185600",
		"ID_PART_ENTRY_SIZE":   "892028928",
		"ID_PART_ENTRY_DISK":   "259:0",
	}
	return data, result
}

func TestParseRunUdevDataFile(t *testing.T) {
	case1Data, case1Result := getCase1DataResult()
	case2Data, case2Result := getCase2DataResult()
	case3Data, case3Result := getCase3DataResult()
	case4Data, case4Result := getCase4DataResult()
	case5Data, case5Result := getCase5DataResult()
	case6Data, case6Result := getCase6DataResult()
	case7Data, case7Result := getCase7DataResult()
	case8Data, case8Result := getCase8DataResult()

	testCases := []struct {
		data           string
		expectedResult map[string]string
	}{
		{case1Data, case1Result},
		{case2Data, case2Result},
		{case3Data, case3Result},
		{case4Data, case4Result},
		{case5Data, case5Result},
		{case6Data, case6Result},
		{case7Data, case7Result},
		{case8Data, case8Result},
	}

	for i, testCase := range testCases {
		result, err := parseRunUdevDataFile(strings.NewReader(testCase.data))
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			for key := range result {
				if result[key] != testCase.expectedResult[key] {
					fmt.Println(key, result[key], testCase.expectedResult[key])
				}
			}
			t.Fatalf("case %v: expected: %+v, got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestNewDevice(t *testing.T) {
	_, case1Event := getCase1DataResult()
	case1Result := &Device{
		Name:    "zram0",
		Major:   252,
		Minor:   0,
		Virtual: true,
	}

	_, case2Event := getCase2DataResult()
	case2Result := &Device{
		Name:         "sda",
		Major:        8,
		Minor:        0,
		Model:        "Ultra",
		UeventSerial: "4C531234567891234567",
		Vendor:       "SanDisk",
		PTType:       "dos",
	}

	_, case3Event := getCase3DataResult()
	case3Result := &Device{
		Name:         "sda1",
		Major:        8,
		Minor:        1,
		Partition:    1,
		Model:        "Ultra",
		UeventSerial: "4C531234567891234567",
		Vendor:       "SanDisk",
		PTType:       "dos",
		UeventFSUUID: "1234-ABCD",
		FSType:       "vfat",
		FSUUID:       "1234-ABCD",
	}

	_, case4Event := getCase4DataResult()
	case4Result := &Device{
		Name:         "nvme0n1",
		Major:        259,
		Minor:        0,
		WWID:         "eui.000000000000000200a012304bd678b5",
		Model:        "Micron 2300 NVMe 512GB",
		UeventSerial: "12345ABCD678",
		PTUUID:       "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		PTType:       "gpt",
	}

	_, case5Event := getCase5DataResult()
	case5Result := &Device{
		Name:         "nvme0n1p1",
		Major:        259,
		Minor:        1,
		Partition:    1,
		WWID:         "eui.000000000000000200a012304bd678b5",
		Model:        "Micron 2300 NVMe 512GB",
		UeventSerial: "12345ABCD678",
		PTUUID:       "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		PTType:       "gpt",
		PartUUID:     "9a69a545-28c3-441c-a60b-6ed5223b03c3",
		UeventFSUUID: "4321-FEDC",
		FSType:       "vfat",
		FSUUID:       "4321-FEDC",
	}

	_, case6Event := getCase6DataResult()
	case6Result := &Device{
		Name:         "nvme0n1p2",
		Major:        259,
		Minor:        2,
		Partition:    2,
		WWID:         "eui.000000000000000200a012304bd678b5",
		Model:        "Micron 2300 NVMe 512GB",
		UeventSerial: "12345ABCD678",
		PTUUID:       "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		PTType:       "gpt",
		PartUUID:     "0959536f-134a-477f-9c02-a7916d034a33",
		UeventFSUUID: "9b7c849b-387e-43f8-ad5a-b7d68c5c062f",
		FSType:       "ext4",
		FSUUID:       "9b7c849b-387e-43f8-ad5a-b7d68c5c062f",
	}

	_, case7Event := getCase7DataResult()
	case7Result := &Device{
		Name:         "nvme0n1p3",
		Major:        259,
		Minor:        3,
		Partition:    3,
		WWID:         "eui.000000000000000200a012304bd678b5",
		Model:        "Micron 2300 NVMe 512GB",
		UeventSerial: "12345ABCD678",
		PTUUID:       "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		PTType:       "gpt",
		PartUUID:     "f8b65530-58bb-4464-b257-d3cb2aba034b",
		UeventFSUUID: "1c9fee93-cc76-4d9d-a1b1-9895c06df6e3",
		FSType:       "ext4",
		FSUUID:       "1c9fee93-cc76-4d9d-a1b1-9895c06df6e3",
	}

	_, case8Event := getCase8DataResult()
	case8Result := &Device{
		Name:         "nvme0n1p4",
		Major:        259,
		Minor:        4,
		Partition:    4,
		WWID:         "eui.000000000000000200a012304bd678b5",
		Model:        "Micron 2300 NVMe 512GB",
		UeventSerial: "12345ABCD678",
		PTUUID:       "27c9e87c-45b2-44eb-b0be-cf52b7d47794",
		PTType:       "gpt",
		PartUUID:     "656b4db9-e2f9-42a6-b09e-9466dcb070dc",
		UeventFSUUID: "a49f8e69-03fb-4735-b900-91d068fcbb70",
		FSType:       "ext4",
		FSUUID:       "a49f8e69-03fb-4735-b900-91d068fcbb70",
	}

	testCases := []struct {
		event          map[string]string
		name           string
		major          int
		minor          int
		virtual        bool
		expectedResult *Device
	}{
		{case1Event, "zram0", 252, 0, true, case1Result},
		{case2Event, "sda", 8, 0, false, case2Result},
		{case3Event, "sda1", 8, 1, false, case3Result},
		{case4Event, "nvme0n1", 259, 0, false, case4Result},
		{case5Event, "nvme0n1p1", 259, 1, false, case5Result},
		{case6Event, "nvme0n1p2", 259, 2, false, case6Result},
		{case7Event, "nvme0n1p3", 259, 3, false, case7Result},
		{case8Event, "nvme0n1p4", 259, 4, false, case8Result},
	}

	for i, testCase := range testCases {
		result, err := NewDevice(testCase.event, testCase.name, testCase.major, testCase.minor, testCase.virtual)
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %+v, got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestParseCDROMs(t *testing.T) {
	case1Data := `CD-ROM information, Id: cdrom.c 3.20 2003/12/17

drive name:	
drive speed:	
drive # of slots:
Can close tray:	
Can open tray:	
Can lock tray:	
Can change speed:
Can select disk:
Can read multisession:
Can read MCN:	
Reports media changed:
Can play audio:	
Can write CD-R:	
Can write CD-RW:
Can read DVD:	
Can write DVD-R:
Can write DVD-RAM:
Can read MRW:	
Can write MRW:	
Can write RAM:	


`
	case2Data := `CD-ROM information, Id: cdrom.c 3.20 2003/12/17

drive name:		sr0
drive speed:		4
drive # of slots:	1
Can close tray:		1
Can open tray:		1
Can lock tray:		1
Can change speed:	1
Can select disk:	0
Can read multisession:	1
Can read MCN:		1
Reports media changed:	1
Can play audio:		1
Can write CD-R:		0
Can write CD-RW:	0
Can read DVD:		1
Can write DVD-R:	0
Can write DVD-RAM:	0
Can read MRW:		1
Can write MRW:		1
Can write RAM:		1


`
	case3Data := `CD-ROM information, Id: cdrom.c 3.20 2003/12/17

drive name:		sr1	sr0
drive speed:		4	50
drive # of slots:	1	1
Can close tray:		1	1
Can open tray:		1	1
Can lock tray:		1	1
Can change speed:	1	1
Can select disk:	0	0
Can read multisession:	1	1
Can read MCN:		1	1
Reports media changed:	1	1
Can play audio:		1	1
Can write CD-R:		0	0
Can write CD-RW:	0	0
Can read DVD:		1	1
Can write DVD-R:	0	0
Can write DVD-RAM:	0	0
Can read MRW:		1	1
Can write MRW:		1	1
Can write RAM:		1	1


`
	testCases := []struct {
		data           string
		expectedResult map[string]struct{}
	}{
		{case1Data, map[string]struct{}{}},
		{case2Data, map[string]struct{}{"sr0": {}}},
		{case3Data, map[string]struct{}{"sr0": {}, "sr1": {}}},
	}

	for i, testCase := range testCases {
		result, err := parseCDROMs(strings.NewReader(testCase.data))
		if err != nil {
			t.Fatalf("case %v: unexpected error %v", i+1, err)
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %+v, got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}
