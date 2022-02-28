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
type matchFn func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error)

type matchResult string

const (
	noMatch        matchResult = "nomatch"
	tooManyMatches matchResult = "toomanymatches"
	changed        matchResult = "changed"
	noChange       matchResult = "nochange"
)

// prioritiy based matchers
var matchers = []matchFn{
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
	filesystemMatcher,
	ueventFSUUIDMatcher,
	// legacy matchers
	fsUUIDMatcher,
	serialNumberMatcher,
	// size matchers
	logicalBlocksizeMatcher,
	physicalBlocksizeMatcher,
	totalCapacityMatcher,
	allocatedCapacityMatcher,
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
func runMatcher(drives []*directcsi.DirectCSIDrive, device *sys.Device) (*directcsi.DirectCSIDrive, matchResult) {
	var matchedDrives, consideredDrives []*directcsi.DirectCSIDrive
	var matched, updated bool
	for _, matchFn := range matchers {
		if len(drives) == 0 {
			break
		}
		matchedDrives, consideredDrives = match(drives, device, matchFn)
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

	switch len(drives) {
	case 0:
		return nil, noMatch
	case 1:
		if updated {
			return drives[0], changed
		}
		return drives[0], noChange
	default:
		// handle too many matches
		//
		// case 1: It is possible to have an empty/partial drive (&directCSIDrive.Status{Path: /dev/sdb0, "", "", "", ...})
		//         to match  with a correct match

		// case 2: A terminating drive and an actual drive can be matched with the single device
		//
		// case 3: A duplicate drive (due to any bug)
		//
		// ToDo: make these drives invalid / decide based on drive status / calculate ranks and decide
		return nil, tooManyMatches
	}
}

// ------------------------
// - Run the list of drives over the provided match function
// - If matched, append the drive to the matched list
// - If considered, append the the drive to the considered list
// ------------------------
func match(drives []*directcsi.DirectCSIDrive, device *sys.Device, matchFn matchFn) ([]*directcsi.DirectCSIDrive, []*directcsi.DirectCSIDrive) {
	var matchedDrives, consideredDrives []*directcsi.DirectCSIDrive
	for _, drive := range drives {
		match, consider, err := matchFn(drive, device)
		if err != nil {
			klog.V(3).Infof("error while matching drive %s: %v", device.DevPath(), err)
			continue
		}
		if match {
			matchedDrives = append(matchedDrives, drive)
		} else if consider {
			consideredDrives = append(consideredDrives, drive)
		}
	}
	return matchedDrives, consideredDrives
}

func fsUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func ueventFSUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func serialNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func ueventSerialNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func wwidMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func modelNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func vendorMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func partitionNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func dmUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func mdUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func partitionUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func partitionTableUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func logicalBlocksizeMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func physicalBlocksizeMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func filesystemMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func totalCapacityMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}

func allocatedCapacityMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, consider bool, err error) {
	// To-Do: impelement matcher function
	return
}
