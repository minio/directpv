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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
)

func TestPartitionNumberMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionNum: 0}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionNum: 1}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionNum: 2}}
	case1Device := &sys.Device{Partition: 0}
	case2Device := &sys.Device{Partition: 1}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		{case1Device, &case1Drive, true, false, nil},
		{case1Device, &case2Drive, false, false, nil},
		{case1Device, &case3Drive, false, false, nil},
		{case2Device, &case2Drive, true, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := partitionNumberMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestUeventSerialNumberMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventSerial: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventSerial: "serial"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventSerial: "serial123"}}
	case1Device := &sys.Device{UeventSerial: ""}
	case2Device := &sys.Device{UeventSerial: "serial"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// UeventSerial blank in both
		{case1Device, &case1Drive, false, true, nil},
		// UeventSerial blank in device
		{case1Device, &case2Drive, false, false, nil},
		// UeventSerial blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// UeventSerial not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// UeventSerial not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := ueventSerialNumberMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestWWIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{WWID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{WWID: "wwid"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{WWID: "wwid123"}}
	case1Device := &sys.Device{WWID: ""}
	case2Device := &sys.Device{WWID: "wwid"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// WWID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// WWID blank in device
		{case1Device, &case2Drive, false, false, nil},
		// WWID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// WWID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// WWID not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := wwidMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestModelNumberMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{ModelNumber: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{ModelNumber: "KXG6AZNV512G TOSHIBA"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{ModelNumber: "KXG6AZ DELL"}}
	case1Device := &sys.Device{Model: ""}
	case2Device := &sys.Device{Model: "KXG6AZNV512G TOSHIBA"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// ModelNumber blank in both
		{case1Device, &case1Drive, false, true, nil},
		// ModelNumber blank in device
		{case1Device, &case2Drive, false, false, nil},
		// ModelNumber blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// ModelNumber not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// ModelNumber not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := modelNumberMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestVendorMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Vendor: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Vendor: "TOSHIBA"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Vendor: "DELL"}}
	case1Device := &sys.Device{Vendor: ""}
	case2Device := &sys.Device{Vendor: "TOSHIBA"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// Vendor blank in both
		{case1Device, &case1Drive, false, true, nil},
		// Vendor blank in device
		{case1Device, &case2Drive, false, false, nil},
		// Vendor blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// Vendor not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// Vendor not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := vendorMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestPartitionUUIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionUUID: "ptuuid"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartitionUUID: "invalidptuuid"}}
	case1Device := &sys.Device{PartUUID: ""}
	case2Device := &sys.Device{PartUUID: "ptuuid"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// PartitionUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// PartitionUUID blank in device
		{case1Device, &case2Drive, false, false, nil},
		// PartitionUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// PartitionUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// PartitionUUID not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := partitionUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestDMUUIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{DMUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{DMUUID: "TOSHIBA"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{DMUUID: "DELL"}}
	case1Device := &sys.Device{DMUUID: ""}
	case2Device := &sys.Device{DMUUID: "TOSHIBA"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// DMUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// DMUUID blank in device
		{case1Device, &case2Drive, false, false, nil},
		// DMUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// DMUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// DMUUID not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := dmUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestMDUUIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{MDUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{MDUUID: "TOSHIBA"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{MDUUID: "DELL"}}
	case1Device := &sys.Device{MDUUID: ""}
	case2Device := &sys.Device{MDUUID: "TOSHIBA"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// MDUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// MDUUID blank in device
		{case1Device, &case2Drive, false, false, nil},
		// MDUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// MDUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// MDUUID not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := mdUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestSerialNumberMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{SerialNumber: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{SerialNumber: "31IF73XDFDM3"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{SerialNumber: "different-31IF73XDFDM3"}}
	case1Device := &sys.Device{Serial: ""}
	case2Device := &sys.Device{Serial: "31IF73XDFDM3"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// SerialNumber blank in both
		{case1Device, &case1Drive, false, true, nil},
		// SerialNumber blank in device
		{case1Device, &case2Drive, false, false, nil},
		// SerialNumber blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// SerialNumber not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// SerialNumber not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := serialNumberMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestUeventFSUUIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventFSUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventFSUUID: "ueventfsuuid"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{UeventFSUUID: "invalid-ueventfsuuid"}}
	case1Device := &sys.Device{UeventFSUUID: ""}
	case2Device := &sys.Device{UeventFSUUID: "ueventfsuuid"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// UeventFSUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// UeventFSUUID blank in device
		{case1Device, &case2Drive, false, true, nil},
		// UeventFSUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// UeventFSUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// UeventFSUUID not blank in both and does not match
		{case2Device, &case3Drive, false, true, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := ueventFSUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestFileSystemTypeMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Filesystem: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Filesystem: "xfs"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{Filesystem: "ext64"}}
	case1Device := &sys.Device{FSType: ""}
	case2Device := &sys.Device{FSType: "xfs"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// Filesystem blank in both
		{case1Device, &case1Drive, false, true, nil},
		// Filesystem blank in device
		{case1Device, &case2Drive, false, true, nil},
		// Filesystem blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// Filesystem not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// Filesystem not blank in both and does not match
		{case2Device, &case3Drive, false, true, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := fileSystemTypeMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestFSUUIDMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{FilesystemUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{FilesystemUUID: "fsuuid"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{FilesystemUUID: "invalid-fsuuid"}}
	case1Device := &sys.Device{FSUUID: ""}
	case2Device := &sys.Device{FSUUID: "fsuuid"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// FilesystemUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// FilesystemUUID blank in device
		{case1Device, &case2Drive, false, true, nil},
		// FilesystemUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// FilesystemUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// FilesystemUUID not blank in both and does not match
		{case2Device, &case3Drive, false, true, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := fsUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestLogicalBlocksizeMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{LogicalBlockSize: 0}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{LogicalBlockSize: 512}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{LogicalBlockSize: -123}}
	case1Device := &sys.Device{LogicalBlockSize: 0}
	case2Device := &sys.Device{LogicalBlockSize: 512}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// LogicalBlockSize blank in both
		{case1Device, &case1Drive, false, true, nil},
		// LogicalBlockSize blank in device
		{case1Device, &case2Drive, false, true, nil},
		// LogicalBlockSize blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// LogicalBlockSize not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// LogicalBlockSize not blank in both and does not match
		{case2Device, &case3Drive, false, true, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := logicalBlocksizeMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestTotalCapacityMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{TotalCapacity: 0}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{TotalCapacity: 512}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{TotalCapacity: -123}}
	case1Device := &sys.Device{TotalCapacity: 0}
	case2Device := &sys.Device{TotalCapacity: 512}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// FilesystemUUID blank in both
		{case1Device, &case1Drive, false, true, nil},
		// FilesystemUUID blank in device
		{case1Device, &case2Drive, false, true, nil},
		// FilesystemUUID blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// FilesystemUUID not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// FilesystemUUID not blank in both and does not match
		{case2Device, &case3Drive, false, true, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := totalCapacityMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestPhysicalBlocksizeMatcher(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PhysicalBlockSize: 0}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PhysicalBlockSize: 512}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PhysicalBlockSize: -123}}
	case1Device := &sys.Device{PhysicalBlockSize: 0}
	case2Device := &sys.Device{PhysicalBlockSize: 512}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// PhysicalBlockSize blank in both
		{case1Device, &case1Drive, false, true, nil},
		// PhysicalBlockSize blank in device
		{case1Device, &case2Drive, false, true, nil},
		// PhysicalBlockSize blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// PhysicalBlockSize not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// PhysicalBlockSize not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := physicalBlocksizeMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}

func TestPartitionTableUUIDMatcherr(t *testing.T) {
	case1Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartTableUUID: ""}}
	case2Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartTableUUID: "serial"}}
	case3Drive := directcsi.DirectCSIDrive{Status: directcsi.DirectCSIDriveStatus{PartTableUUID: "serial123"}}
	case1Device := &sys.Device{PartUUID: ""}
	case2Device := &sys.Device{PartUUID: "serial"}
	testCases := []struct {
		device   *sys.Device
		drive    *directcsi.DirectCSIDrive
		match    bool
		consider bool
		err      error
	}{
		// UeventSerial blank in both
		{case1Device, &case1Drive, false, true, nil},
		// UeventSerial blank in device
		{case1Device, &case2Drive, false, false, nil},
		// UeventSerial blank in drive
		{case2Device, &case1Drive, false, true, nil},
		// UeventSerial not blank in both and match
		{case2Device, &case2Drive, true, false, nil},
		// UeventSerial not blank in both and does not match
		{case2Device, &case3Drive, false, false, nil},
	}

	for i, testCase := range testCases {
		match, consider, err := partitionTableUUIDMatcher(testCase.device, testCase.drive)
		if match != testCase.match || consider != testCase.consider || err != testCase.err {
			t.Fatalf("case %v: expected: match %v , consider %v , error %v ; got: match %v  consider %v  error %v ", i+1, match, consider, err, testCase.match, testCase.consider, testCase.err)
		}
	}
}
