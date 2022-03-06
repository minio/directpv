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

package node

import (
	"context"
	"errors"
	"reflect"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsDOSPTType(t *testing.T) {
	testCases := []struct {
		ptType         string
		expectedResult bool
	}{
		{"dos", true},
		{"msdos", true},
		{"mbr", true},
		{"gpt", false},
	}

	for i, testCase := range testCases {
		result := isDOSPTType(testCase.ptType)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestPTTypeEqual(t *testing.T) {
	testCases := []struct {
		ptType1        string
		ptType2        string
		expectedResult bool
	}{
		{"dos", "dos", true},
		{"msdos", "mbr", true},
		{"gpt", "gpt", true},
		{"mbr", "gpt", false},
	}

	for i, testCase := range testCases {
		result := ptTypeEqual(testCase.ptType1, testCase.ptType2)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsHWInfoAvailable(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		WWID: "eui.000000000000000200a012304bd678b5",
	}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		SerialNumber: "4C531234567891234567",
	}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		UeventSerial: "12345ABCD678",
	}}
	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		WWID:         "eui.000000000000000200a012304bd678b5",
		SerialNumber: "4C531234567891234567",
		UeventSerial: "12345ABCD678",
	}}
	case5Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{case1Drive, true},
		{case2Drive, true},
		{case3Drive, true},
		{case4Drive, true},
		{case5Drive, false},
	}

	for i, testCase := range testCases {
		result := isHWInfoAvailable(&testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDeviceHWInfo(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		WWID: "eui.000000000000000200a012304bd678b5",
	}}
	case1Device := &sys.Device{WWID: "eui.000000000000000200a012304bd678b5"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		SerialNumber: "4C531234567891234567",
	}}
	case2Device := &sys.Device{Serial: "4C531234567891234567"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		UeventSerial: "12345ABCD678",
	}}
	case3Device := &sys.Device{UeventSerial: "12345ABCD678"}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		WWID:         "eui.000000000000000200a012304bd678b5",
		SerialNumber: "4C531234567891234567",
		UeventSerial: "12345ABCD678",
	}}
	case4Device := &sys.Device{
		WWID:         "eui.000000000000000200a012304bd678b5",
		Serial:       "4C531234567891234567",
		UeventSerial: "12345ABCD678",
	}

	case5Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case5Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case5Drive, case5Device, true},
		{case5Drive, case4Device, true},
		{case1Drive, case2Device, false},
		{case1Drive, case5Device, false},
	}

	for i, testCase := range testCases {
		result := matchDeviceHWInfo(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsDMMDUUIDAvailable(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{case1Drive, true},
		{case2Drive, true},
		{case3Drive, true},
		{case4Drive, false},
	}

	for i, testCase := range testCases {
		result := isDMMDUUIDAvailable(&testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDeviceDMMDUUID(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case1Device := &sys.Device{DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case2Device := &sys.Device{MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case3Device := &sys.Device{
		DMUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		MDUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case4Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case4Drive, case1Device, true},
		{case4Drive, case3Device, true},
		{case1Drive, case2Device, false},
		{case1Drive, case4Device, false},
	}

	for i, testCase := range testCases {
		result := matchDeviceDMMDUUID(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsPTUUIDAvailable(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  0,
		PartTableUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  1,
		PartTableUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{case1Drive, true},
		{case2Drive, false},
		{case3Drive, false},
	}

	for i, testCase := range testCases {
		result := isPTUUIDAvailable(&testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDevicePTUUID(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  0,
		PartTableUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case1Device := &sys.Device{Partition: 0, PTUUID: "2ac7a498-1859-4815-8864-41890803cb0b"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  0,
		PartTableUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		PartTableType: "gpt",
	}}
	case2Device := &sys.Device{Partition: 0, PTUUID: "2ac7a498-1859-4815-8864-41890803cb0b", PTType: "gpt"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  1,
		PartTableUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		PartTableType: "dos",
	}}
	case3Device := &sys.Device{Partition: 1, PTUUID: "2ac7a498-1859-4815-8864-41890803cb0b", PTType: "dos"}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case4Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case1Drive, case2Device, false},
		{case1Drive, case3Device, false},
		{case3Drive, case4Device, false},
	}

	for i, testCase := range testCases {
		result := matchDevicePTUUID(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsPartUUIDAvailable(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  1,
		PartitionUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  0,
		PartitionUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum: 1,
	}}
	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{case1Drive, true},
		{case2Drive, false},
		{case3Drive, false},
		{case4Drive, false},
	}

	for i, testCase := range testCases {
		result := isPartUUIDAvailable(&testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDevicePartUUID(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  1,
		PartitionUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case1Device := &sys.Device{Partition: 1, PartUUID: "2ac7a498-1859-4815-8864-41890803cb0b"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case2Device := &sys.Device{PartUUID: "2ac7a498-1859-4815-8864-41890803cb0b"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum: 1,
	}}
	case3Device := &sys.Device{Partition: 1}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case4Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case1Drive, case2Device, false},
	}

	for i, testCase := range testCases {
		result := matchDevicePartUUID(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestIsFSUUIDAvailable(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		FilesystemUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		UeventFSUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		FilesystemUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		UeventFSUUID:   "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
	}}
	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		expectedResult bool
	}{
		{case1Drive, true},
		{case2Drive, true},
		{case3Drive, true},
		{case4Drive, false},
	}

	for i, testCase := range testCases {
		result := isFSUUIDAvailable(&testCase.drive)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDeviceFSUUID(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		FilesystemUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
	}}
	case1Device := &sys.Device{FSUUID: "2ac7a498-1859-4815-8864-41890803cb0b"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		FilesystemUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		Filesystem:     "xfs",
	}}
	case2Device := &sys.Device{FSUUID: "2ac7a498-1859-4815-8864-41890803cb0b", FSType: "xfs"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		UeventFSUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
		Filesystem:   "ext4",
	}}
	case3Device := &sys.Device{UeventFSUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec", FSType: "ext4"}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		FilesystemUUID: "2ac7a498-1859-4815-8864-41890803cb0b",
		UeventFSUUID:   "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
		Filesystem:     "xfs",
	}}
	case4Device := &sys.Device{
		FSUUID:       "2ac7a498-1859-4815-8864-41890803cb0b",
		UeventFSUUID: "92df2dae-8c4f-4a62-9dab-eec1a0d627ec",
		FSType:       "xfs",
	}

	case5Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case5Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case5Drive, case5Device, true},
		{case1Drive, case2Device, false},
	}

	for i, testCase := range testCases {
		result := matchDeviceFSUUID(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestMatchDeviceNameSize(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		TotalCapacity: 500107608,
		Path:          "/dev/sda",
	}}
	case1Device := &sys.Device{Size: 500107608, Name: "sda"}

	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		Virtual:       true,
		TotalCapacity: 500107608,
		Path:          "/dev/sda",
	}}
	case2Device := &sys.Device{Virtual: true, Size: 500107608, Name: "sda"}

	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		ReadOnly:      true,
		TotalCapacity: 500107608,
		Path:          "/dev/sda",
	}}
	case3Device := &sys.Device{ReadOnly: true, Size: 500107608, Name: "sda"}

	case4Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		PartitionNum:  1,
		Virtual:       true,
		ReadOnly:      true,
		TotalCapacity: 500107608,
		Path:          "/dev/sda1",
	}}
	case4Device := &sys.Device{Partition: 1, Virtual: true, ReadOnly: true, Size: 500107608, Name: "sda1"}

	case5Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{}}
	case5Device := &sys.Device{}

	testCases := []struct {
		drive          directcsi.DirectCSIDrive
		device         *sys.Device
		expectedResult bool
	}{
		{case1Drive, case1Device, true},
		{case2Drive, case2Device, true},
		{case3Drive, case3Device, true},
		{case4Drive, case4Device, true},
		{case5Drive, case5Device, false},
		{case1Drive, case2Device, false},
		{case2Drive, case3Device, false},
	}

	for i, testCase := range testCases {
		result := matchDeviceNameSize(&testCase.drive, testCase.device)
		if result != testCase.expectedResult {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult, result)
		}
	}
}

func TestUpdateDriveProperties(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		AllocatedCapacity: 5120000,
		Virtual:           true,
		ReadOnly:          true,
		Partitioned:       true,
		SwapOn:            true,
	}}
	case1Device := &sys.Device{
		FSType:            "xfs",
		Size:              512000,
		LogicalBlockSize:  1024,
		Model:             "QEMU",
		FirstMountPoint:   "/data",
		FirstMountOptions: []string{"rw"},
		Partition:         1,
		Name:              "vda1",
		PhysicalBlockSize: 512,
		Serial:            "1A2B3C4D",
		FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
		PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
		Major:             8,
		Minor:             1,
		UeventSerial:      "12345ABCD678",
		UeventFSUUID:      "a7ef53ca-a96f-4dc1-a201-3c086d7f4962",
		WWID:              "ABCD000000001234567",
		Vendor:            "KVM",
		DMName:            "vg0-lv0",
		DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
		MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
		PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
		PTType:            "gpt",
		Master:            "vda",
	}
	case1Result := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{
		Filesystem:        "xfs",
		TotalCapacity:     512000,
		LogicalBlockSize:  1024,
		ModelNumber:       "QEMU",
		Mountpoint:        "/data",
		MountOptions:      []string{"rw"},
		PartitionNum:      1,
		PhysicalBlockSize: 512,
		Path:              "/dev/vda1",
		RootPartition:     "vda1",
		SerialNumber:      "1A2B3C4D",
		FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
		PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
		MajorNumber:       8,
		MinorNumber:       1,
		UeventSerial:      "12345ABCD678",
		UeventFSUUID:      "a7ef53ca-a96f-4dc1-a201-3c086d7f4962",
		WWID:              "ABCD000000001234567",
		Vendor:            "KVM",
		DMName:            "vg0-lv0",
		DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
		MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
		PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
		PartTableType:     "gpt",
		Master:            "vda",
	}}

	testCases := []struct {
		drive           directcsi.DirectCSIDrive
		device          *sys.Device
		expectedResult  directcsi.DirectCSIDrive
		expectedUpdated bool
	}{
		{case1Drive, case1Device, case1Result, true},
		{case1Result, case1Device, directcsi.DirectCSIDrive{}, false},
	}

	for i, testCase := range testCases {
		drive := testCase.drive
		updated, _ := updateDriveProperties(&drive, testCase.device)
		result := drive
		if updated != testCase.expectedUpdated {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedUpdated, updated)
		}

		if !updated {
			continue
		}

		if result.Status.Filesystem != testCase.expectedResult.Status.Filesystem {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.Filesystem, result.Status.Filesystem)
		}
		if result.Status.TotalCapacity != testCase.expectedResult.Status.TotalCapacity {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.TotalCapacity, result.Status.TotalCapacity)
		}
		if result.Status.LogicalBlockSize != testCase.expectedResult.Status.LogicalBlockSize {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.LogicalBlockSize, result.Status.LogicalBlockSize)
		}
		if result.Status.ModelNumber != testCase.expectedResult.Status.ModelNumber {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.ModelNumber, result.Status.ModelNumber)
		}
		if result.Status.Mountpoint != testCase.expectedResult.Status.Mountpoint {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.Mountpoint, result.Status.Mountpoint)
		}
		if !reflect.DeepEqual(result.Status.MountOptions, testCase.expectedResult.Status.MountOptions) {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.MountOptions, result.Status.MountOptions)
		}
		if result.Status.PartitionNum != testCase.expectedResult.Status.PartitionNum {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.PartitionNum, result.Status.PartitionNum)
		}
		if result.Status.PhysicalBlockSize != testCase.expectedResult.Status.PhysicalBlockSize {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.PhysicalBlockSize, result.Status.PhysicalBlockSize)
		}
		if result.Status.Path != testCase.expectedResult.Status.Path {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.Path, result.Status.Path)
		}
		if result.Status.RootPartition != testCase.expectedResult.Status.RootPartition {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.RootPartition, result.Status.RootPartition)
		}
		if result.Status.SerialNumber != testCase.expectedResult.Status.SerialNumber {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.SerialNumber, result.Status.SerialNumber)
		}
		if result.Status.FilesystemUUID != testCase.expectedResult.Status.FilesystemUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.FilesystemUUID, result.Status.FilesystemUUID)
		}
		if result.Status.PartitionUUID != testCase.expectedResult.Status.PartitionUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.PartitionUUID, result.Status.PartitionUUID)
		}
		if result.Status.MajorNumber != testCase.expectedResult.Status.MajorNumber {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.MajorNumber, result.Status.MajorNumber)
		}
		if result.Status.MinorNumber != testCase.expectedResult.Status.MinorNumber {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.MinorNumber, result.Status.MinorNumber)
		}
		if result.Status.UeventSerial != testCase.expectedResult.Status.UeventSerial {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.UeventSerial, result.Status.UeventSerial)
		}
		if result.Status.UeventFSUUID != testCase.expectedResult.Status.UeventFSUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.UeventFSUUID, result.Status.UeventFSUUID)
		}
		if result.Status.WWID != testCase.expectedResult.Status.WWID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.WWID, result.Status.WWID)
		}
		if result.Status.Vendor != testCase.expectedResult.Status.Vendor {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.Vendor, result.Status.Vendor)
		}
		if result.Status.DMName != testCase.expectedResult.Status.DMName {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.DMName, result.Status.DMName)
		}
		if result.Status.DMUUID != testCase.expectedResult.Status.DMUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.DMUUID, result.Status.DMUUID)
		}
		if result.Status.MDUUID != testCase.expectedResult.Status.MDUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.MDUUID, result.Status.MDUUID)
		}
		if result.Status.PartTableUUID != testCase.expectedResult.Status.PartTableUUID {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.PartTableUUID, result.Status.PartTableUUID)
		}
		if result.Status.PartTableType != testCase.expectedResult.Status.PartTableType {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.PartTableType, result.Status.PartTableType)
		}
		if result.Status.Master != testCase.expectedResult.Status.Master {
			t.Fatalf("case %v: expected: %v; got: %v", i+1, testCase.expectedResult.Status.Master, result.Status.Master)
		}
	}
}

func TestMountDrive(t *testing.T) {
	testCases := []struct {
		drive                    *directcsi.DirectCSIDrive
		mountFn                  func(device, target string, flags []string) error
		expectedMountpoint       string
		expectedInitializedState metav1.ConditionStatus
	}{
		// Mounted already
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "fsuuid",
					Path:           "/dev/path",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:               string(directcsi.DirectCSIDriveConditionReady),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(directcsi.DirectCSIDriveReasonReady),
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			},
			mountFn: func(device, target string, flags []string) error {
				return nil
			},
			expectedMountpoint:       "/var/lib/direct-csi/mnt/fsuuid",
			expectedInitializedState: metav1.ConditionTrue,
		},
		// Successfully mounted
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "fsuuid",
					Path:           "/dev/path",
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionFalse,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:               string(directcsi.DirectCSIDriveConditionReady),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(directcsi.DirectCSIDriveReasonReady),
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			},
			mountFn: func(device, target string, flags []string) error {
				return nil
			},
			expectedMountpoint:       "/var/lib/direct-csi/mnt/fsuuid",
			expectedInitializedState: metav1.ConditionTrue,
		},
		// Mount failed
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					FilesystemUUID: "fsuuid",
					Path:           "/dev/path",
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:               string(directcsi.DirectCSIDriveConditionReady),
							Status:             metav1.ConditionTrue,
							Message:            "",
							Reason:             string(directcsi.DirectCSIDriveReasonReady),
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			},
			mountFn: func(device, target string, flags []string) error {
				return errors.New("error")
			},
			expectedMountpoint:       "",
			expectedInitializedState: metav1.ConditionFalse,
		},
	}

	ctx := context.TODO()
	for i, testCase := range testCases {
		client.SetLatestDirectCSIDriveInterface(fakedirect.NewSimpleClientset(testCase.drive).DirectV1beta4().DirectCSIDrives())
		mountDrive(ctx, testCase.drive, testCase.mountFn)
		drive, err := client.GetLatestDirectCSIDriveInterface().Get(
			ctx, testCase.drive.Name, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
		)
		if err != nil {
			t.Fatalf("case %v: unexpected error: %v", i+1, err)
		}
		if drive.Status.Mountpoint != testCase.expectedMountpoint {
			t.Fatalf("case %v: expected mountpoint: %v but got: %v", i+1, testCase.expectedMountpoint, drive.Status.Mountpoint)
		}
		if !utils.IsConditionStatus(drive.Status.Conditions, string(directcsi.DirectCSIDriveConditionInitialized), testCase.expectedInitializedState) {
			t.Fatalf("case %v: unexpected initializedstate", i+1)
		}
	}
}
