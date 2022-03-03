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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/minio/directpv/pkg/sys"
)

func mapToUdevData(eventMap map[string]string) (*sys.UDevData, error) {
	path := eventMap["DEVPATH"]
	if path == "" {
		return nil, errInvalidDevPath
	}

	major, err := strconv.Atoi(eventMap["MAJOR"])
	if err != nil {
		return nil, err
	}

	minor, err := strconv.Atoi(eventMap["MINOR"])
	if err != nil {
		return nil, err
	}

	var partition int
	if value, found := eventMap["ID_PART_ENTRY_NUMBER"]; found && value != "" {
		partition, err = strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
	}

	return &sys.UDevData{
		Path:         path,
		Major:        major,
		Minor:        minor,
		Partition:    partition,
		WWID:         eventMap["ID_WWN"],
		Model:        eventMap["ID_MODEL"],
		UeventSerial: eventMap["ID_SERIAL_SHORT"],
		Vendor:       eventMap["ID_VENDOR"],
		DMName:       eventMap["DM_NAME"],
		DMUUID:       eventMap["DM_UUID"],
		MDUUID:       sys.NormalizeUUID(eventMap["MD_UUID"]),
		PTUUID:       eventMap["ID_PART_TABLE_UUID"],
		PTType:       eventMap["ID_PART_TABLE_TYPE"],
		PartUUID:     eventMap["ID_PART_ENTRY_UUID"],
		UeventFSUUID: eventMap["ID_FS_UUID"],
		FSType:       eventMap["ID_FS_TYPE"],
	}, nil
}

func getRootBlockPath(devName string) string {
	switch {
	case strings.HasPrefix(devName, sys.HostDevRoot):
		return devName
	case strings.Contains(devName, sys.DirectCSIDevRoot):
		return getRootBlockPath(filepath.Base(devName))
	default:
		name := strings.ReplaceAll(
			strings.Replace(devName, sys.DirectCSIPartitionInfix, "", 1),
			sys.DirectCSIPartitionInfix,
			sys.HostPartitionInfix,
		)
		return filepath.Join(sys.HostDevRoot, name)
	}
}
