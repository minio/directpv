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
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
	"k8s.io/klog/v2"
)

// If the corresponding field is empty in the drive, return consider
// Else, return True if matched or False if not matched
type matchFn func(device *sys.Device, drive *directcsi.DirectCSIDrive) (match bool, consider bool, err error)

type matchResult string

const (
	noMatch        matchResult = "nomatch"
	tooManyMatches matchResult = "toomanymatches"
	changed        matchResult = "changed"
	noChange       matchResult = "nochange"
)

// prioritiy based matchers
var stageOneMatchers = []matchFn{
	// HW matchers
	partitionNumberMatcher,
	ueventSerialNumberMatcher,
	wwidMatcher,
	modelNumberMatcher,
	vendorMatcher,
	// SW matchers
	partitionTableUUIDMatcher,
	partitionUUIDMatcher,
	dmUUIDMatcher,
	mdUUIDMatcher,
	fileSystemTypeMatcher,
	ueventFSUUIDMatcher,
	// v1beta2 matchers
	fsUUIDMatcher,
	serialNumberMatcher,
	// size matchers
	logicalBlocksizeMatcher,
	physicalBlocksizeMatcher,
	totalCapacityMatcher,
}

// stageTwoMatchers are the conslusive matchers for more than one matching drives
var stageTwoMatchers = []matchFn{
	majMinMatcher,
	pathMatcher,
}

// ------------------------
// - Run the list of matchers on the list of drives
// - Match function SHOULD return matched (100% match) and considered (if the field is empty) results
// - If MORE THAN ONE matching drive is found, pass the matching drives to the next matching function in the priority list
//   If NO matching drive is found
//      AND If the considered list is empty, return the empty list (NEW DRIVE).
//          Else, pass the considered list to the next matching function in the priority list
//   (NOTE: the returning results can be more than one incase of duplicate/identical drives)
// ------------------------
func runMatchers(drives []*directcsi.DirectCSIDrive,
	device *sys.Device,
	stageOneMatchers, stageTwoMatchers []matchFn) (*directcsi.DirectCSIDrive, matchResult) {
	var matchedDrives, consideredDrives []*directcsi.DirectCSIDrive
	var matched, updated bool
	var err error

	for _, matchFn := range stageOneMatchers {
		if len(drives) == 0 {
			break
		}
		matchedDrives, consideredDrives, err = match(drives, device, matchFn)
		if err != nil {
			klog.V(3).Infof("error while matching drive %s: %v", device.DevPath(), err)
			continue
		}
		switch {
		case len(matchedDrives) > 0:
			if len(matchedDrives) == 1 {
				matched = true
			}
			drives = matchedDrives
		default:
			if len(consideredDrives) == 1 && matched {
				updated = true
			}
			drives = consideredDrives
		}
	}

	if len(drives) > 1 {
		for _, matchFn := range stageTwoMatchers {
			// run identity matcher for more than one matched drives
			drives, _, err = match(drives, device, matchFn)
			if err != nil {
				klog.V(3).Infof("error while matching drive %s: %v", device.DevPath(), err)
				continue
			}
		}
	}

	switch len(drives) {
	case 0:
		return nil, noMatch
	case 1:
		if updated {
			return drives[0], changed
		}
		return drives[0], noChange
	default:
		return nil, tooManyMatches
	}
}

func majMinMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (match bool, consider bool, err error) {
	if drive.Status.MajorNumber != uint32(device.Major) {
		return false, false, nil
	}
	if drive.Status.MinorNumber != uint32(device.Minor) {
		return false, false, nil
	}
	return true, true, nil
}

func pathMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (match bool, consider bool, err error) {
	if getRootBlockPath(drive.Status.Path) != device.DevPath() {
		return false, false, nil
	}
	return true, true, nil
}

// ------------------------
// - Run the list of drives over the provided match function
// - If matched, append the drive to the matched list
// - If considered, append the the drive to the considered list
// ------------------------
func match(drives []*directcsi.DirectCSIDrive,
	device *sys.Device,
	matchFn matchFn) ([]*directcsi.DirectCSIDrive, []*directcsi.DirectCSIDrive, error) {
	var matchedDrives, consideredDrives []*directcsi.DirectCSIDrive
	for _, drive := range drives {
		if drive.Status.DriveStatus == directcsi.DriveStatusTerminating {
			continue
		}
		match, consider, err := matchFn(device, drive)
		if err != nil {
			return nil, nil, err
		}
		if match {
			matchedDrives = append(matchedDrives, drive)
		} else if consider {
			consideredDrives = append(consideredDrives, drive)
		}
	}
	return matchedDrives, consideredDrives, nil
}

func partitionNumberMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return drive.Status.PartitionNum == device.Partition, false, nil
}

func ueventSerialNumberMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.UeventSerial, drive.Status.UeventSerial)
}

func wwidMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.WWID, drive.Status.WWID)
}

func modelNumberMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.Model, drive.Status.ModelNumber)
}

func vendorMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.Vendor, drive.Status.Vendor)
}

func partitionUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.PartUUID, drive.Status.PartitionUUID)
}

func dmUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.DMUUID, drive.Status.DMUUID)
}

func mdUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.MDUUID, drive.Status.MDUUID)
}

func partitionTableUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return immutablePropertyMatcher(device.PartUUID, drive.Status.PartTableUUID)
}

// Refer https://go.dev/play/p/zuaURPArfcL
// ###################################### Truth table Hardware Matcher ####################################
//	| alpha | beta |				Match 						|			Not-Match 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   0	|   0  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   0	|   1  |				XXXXXXX     	            | match=false, consider=false, err = nil  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   0  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   1  |	match=true, consider=false, err = nil   | match=false, consider=false, err = nil  |
//  |-------|------|--------------------------------------------|-----------------------------------------|

func immutablePropertyMatcher(alpha string, beta string) (bool, bool, error) {
	var match, consider bool
	var err error
	switch {
	case alpha == "" && beta == "":
		consider = true
	case alpha == "" && beta != "":
	case alpha != "" && beta == "":
		consider = true
	case alpha != "" && beta != "":
		if alpha == beta {
			match = true
		}
	}
	return match, consider, err
}

func ueventFSUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return fsPropertyMatcher(device.UeventFSUUID, drive.Status.UeventFSUUID)
}

func fsUUIDMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return fsPropertyMatcher(device.FSUUID, drive.Status.FilesystemUUID)
}

func serialNumberMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return fsPropertyMatcher(device.Serial, drive.Status.SerialNumber)
}

func fileSystemTypeMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return fsPropertyMatcher(device.FSType, drive.Status.Filesystem)
}

// Refer https://go.dev/play/p/zuaURPArfcL
// ###################################### Truth table Hardware Matcher ####################################
//	| alpha | beta |				Match 						|			Not-Match 					  |
//	|-------|------|--------------------------------------------|------------------------------------------
// 	|   0	|   0  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   0	|   1  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   0  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   1  |	match=true, consider=false, err = nil   | match=false, consider=true , err = nil  |
//  |-------|------|--------------------------------------------|------------------------------------------

func fsPropertyMatcher(alpha string, beta string) (bool, bool, error) {
	var match, consider bool
	var err error
	switch {
	case alpha == "" && beta == "":
		consider = true
	case alpha == "" && beta != "":
		consider = true
	case alpha != "" && beta == "":
		consider = true
	case alpha != "" && beta != "":
		if alpha == beta {
			match = true
		} else {
			consider = true
		}
	}
	return match, consider, err
}

func logicalBlocksizeMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return sizeMatcher(int64(device.LogicalBlockSize), drive.Status.LogicalBlockSize)
}

func totalCapacityMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	return sizeMatcher(int64(device.TotalCapacity), drive.Status.TotalCapacity)
}

func sizeMatcher(alpha int64, beta int64) (bool, bool, error) {
	var match, consider bool
	var err error
	switch {
	case alpha == 0 && beta == 0:
		consider = true
	case alpha == 0 && beta != 0:
		consider = true
	case alpha != 0 && beta == 0:
		consider = true
	case alpha != 0 && beta != 0:
		if alpha == beta {
			match = true
		} else {
			consider = true
		}
	}
	return match, consider, err
}

// Refer https://go.dev/play/p/zuaURPArfcL
// ###################################### Truth table Hardware Matcher ####################################
//	| alpha | beta |				Match 						|			Not-Match 					  |
//	|-------|------|--------------------------------------------|------------------------------------------
// 	|   0	|   0  |	match=false, consider=true, err = nil   | 			XXXXXXX 					  |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   0	|   1  |				XXXXXXX           			| match=false, consider=true, err = nil   |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   0  |				XXXXXXX 					| match=false, consider=true, err = nil   |
//	|-------|------|--------------------------------------------|-----------------------------------------|
// 	|   1	|   1  |	match=true, consider=false, err = nil   | match=false, consider=fals , err = nil  |
//  |-------|------|--------------------------------------------|-----------------------------------------|

func physicalBlocksizeMatcher(device *sys.Device, drive *directcsi.DirectCSIDrive) (bool, bool, error) {
	var match, consider bool
	var err error
	switch {
	case int64(device.PhysicalBlockSize) == 0 && drive.Status.PhysicalBlockSize == 0:
		consider = true
	case int64(device.PhysicalBlockSize) == 0 && drive.Status.PhysicalBlockSize != 0:
		consider = true
	case int64(device.PhysicalBlockSize) != 0 && drive.Status.PhysicalBlockSize == 0:
		consider = true
	case int64(device.PhysicalBlockSize) != 0 && drive.Status.PhysicalBlockSize != 0:
		if int64(device.PhysicalBlockSize) == drive.Status.PhysicalBlockSize {
			match = true
		}
	}
	return match, consider, err
}
