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
	"errors"
	"fmt"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
)

var (
	errNoMatchFound = errors.New("no matching drive found")
)

type matchFn func(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool)

var matchers = []matchFn{
	fsUUIDMatcher,
	ueventFSUUIDMatcher,
	serialNumberMatcher,
	ueventSerialNumberMatcher,
	wwidMatcher,
	modelNumberMatcher,
	vendorMatcher,
	partitionNumberMatcher,
	dmUUIDMatcher,
	mdUUIDMatcher,
	partitionUUIDMatcher,
	partitionTableUUIDMatcher,
	logicalBlocksizeMatcher,
	physicalBlocksizeMatcher,
	filesystemMatcher,
	totalCapacityMatcher,
	allocatedCapacityMatcher,
	mountMatcher,
}

func runMatcher(drives []*directcsi.DirectCSIDrive, device *sys.Device) (*directcsi.DirectCSIDrive, error) {
	matchedDrives := getMatchingDrives(drives, device)
	switch len(matchedDrives) {
	case 1:
		return matchedDrives[0], nil
	case 0:
		return nil, errNoMatchFound
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
		return nil, fmt.Errorf("device %s has too many matches", device.Name)
	}
}

func getMatchingDrives(drives []*directcsi.DirectCSIDrive, device *sys.Device) (matchedDrives []*directcsi.DirectCSIDrive) {
	for _, drive := range drives {
		for _, matchFn := range matchers {
			match, cont := matchFn(drive, device)
			if cont {
				continue
			}
			if match {
				matchedDrives = append(matchedDrives, drive)
			}
			break
		}
	}
	return
}

func fsUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func ueventFSUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func serialNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func ueventSerialNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func wwidMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func modelNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func vendorMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func partitionNumberMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func dmUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func mdUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func partitionUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func partitionTableUUIDMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func logicalBlocksizeMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func physicalBlocksizeMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func filesystemMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func totalCapacityMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func allocatedCapacityMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}

func mountMatcher(drive *directcsi.DirectCSIDrive, device *sys.Device) (match bool, cont bool) {
	// To-Do: impelement matcher function
	return
}
