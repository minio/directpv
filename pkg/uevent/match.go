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
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/sys"
)

type matchFn func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool

type matchResult string

const (
	noMatch        matchResult = "nomatch"
	tooManyMatches matchResult = "toomanymatches"
	changed        matchResult = "changed"
	noChange       matchResult = "nochange"
)

func isChanged(device *sys.Device, directCSIDrive *directcsi.DirectCSIDrive) bool {
	return !ValidateUDevInfo(device, directCSIDrive) ||
		!ValidateMountInfo(device, directCSIDrive) ||
		!validateSysInfo(device, directCSIDrive) ||
		!validateDevInfo(device, directCSIDrive)
}

func getMatchedDevicesForManagedDrive(drive *directcsi.DirectCSIDrive, devices []*sys.Device) ([]*sys.Device, []*sys.Device) {
	return getMatchedDevices(
		drive,
		devices,
		func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
			return fsMatcher(drive, device)
		},
	)
}

func getMatchedDevicesForNonManagedDrive(drive *directcsi.DirectCSIDrive, devices []*sys.Device) ([]*sys.Device, []*sys.Device) {
	return getMatchedDevices(
		drive,
		devices,
		func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
			return conclusiveMatcher(drive, device)
		},
	)
}

func getMatchedDevicesForUnidentifiedDrive(drive *directcsi.DirectCSIDrive, devices []*sys.Device) ([]*sys.Device, []*sys.Device) {
	return getMatchedDevices(
		drive,
		devices,
		func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
			return nonConclusiveMatcher(drive, device)
		},
	)
}

func getMatchedDevices(drive *directcsi.DirectCSIDrive, devices []*sys.Device, matchFn matchFn) (matchedDevices, unmatchedDevices []*sys.Device) {
	for _, device := range devices {
		if matchFn(drive, device) {
			matchedDevices = append(matchedDevices, device)
		} else {
			unmatchedDevices = append(unmatchedDevices, device)
		}
	}
	return matchedDevices, unmatchedDevices
}

func getMatchedDrives(drives []*directcsi.DirectCSIDrive, device *sys.Device, matchFn matchFn) (matchedDrives []*directcsi.DirectCSIDrive) {
	for _, drive := range drives {
		if matchFn(drive, device) {
			matchedDrives = append(matchedDrives, drive)
		}
	}
	return matchedDrives
}

func fsMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	if drive.Status.Filesystem == "" || drive.Status.FilesystemUUID == "" {
		return false
	}
	if drive.Status.Filesystem != device.FSType {
		return false
	}
	if drive.Status.FilesystemUUID != device.FSUUID {
		return false
	}
	return true
}

func conclusiveMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	if device.Partition != drive.Status.PartitionNum {
		return false
	}
	if drive.Status.WWID != "" {
		if drive.Status.WWID == device.WWID {
			return true
		}
		// WWID of few drives may show up without extension
		// eg, probed wwid = 0x6002248032bf1752a69bdaee7b0ceb33
		//     wwid in drive CRD = naa.6002248032bf1752a69bdaee7b0ceb33
		return strings.TrimPrefix(device.WWID, "0x") == wwidWithoutExtension(drive.Status.WWID)
	}
	if drive.Status.UeventSerial != "" {
		return drive.Status.UeventSerial == device.UeventSerial
	}
	if drive.Status.SerialNumberLong != "" {
		return drive.Status.SerialNumberLong == device.SerialLong
	}
	if drive.Status.DMUUID != "" {
		return drive.Status.DMUUID == device.DMUUID
	}
	if drive.Status.MDUUID != "" {
		return drive.Status.MDUUID == device.MDUUID
	}
	if drive.Status.ModelNumber != "" && drive.Status.ModelNumber != device.Model {
		return false
	}
	if drive.Status.Vendor != "" && drive.Status.Vendor != device.Vendor {
		return false
	}
	if drive.Status.PartitionNum > 0 && drive.Status.PartitionUUID != "" {
		// PartitionUUID values in versions < 3.0.0 is upper-cased
		return strings.EqualFold(drive.Status.PartitionUUID, device.PartUUID)
	}
	if drive.Status.PartTableUUID != "" {
		return strings.EqualFold(drive.Status.PartTableUUID, device.PTUUID)
	}
	if drive.Status.UeventFSUUID != "" {
		return device.UeventFSUUID == drive.Status.UeventFSUUID
	}
	if drive.Status.FilesystemUUID != "" {
		return device.FSUUID == drive.Status.FilesystemUUID
	}
	return false
}

func nonConclusiveMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
	if getRootBlockPath(drive.Status.Path) != device.DevPath() {
		return false
	}
	if drive.Status.MajorNumber != uint32(device.Major) {
		return false
	}
	if drive.Status.MinorNumber != uint32(device.Minor) {
		return false
	}
	if drive.Status.PCIPath != "" && drive.Status.PCIPath != device.PCIPath {
		return false
	}
	return true
}

func matchDrives(drives []*directcsi.DirectCSIDrive, device *sys.Device) (*directcsi.DirectCSIDrive, matchResult) {
	matchedDrives := getMatchedDrives(drives, device, conclusiveMatcher)
	switch len(matchedDrives) {
	case 0:
		return nil, noMatch
	case 1:
		if IsFormatRequested(matchedDrives[0]) || isChanged(device, matchedDrives[0]) {
			return matchedDrives[0], changed
		}
		return matchedDrives[0], noChange
	default:
		return nil, tooManyMatches
	}
}
