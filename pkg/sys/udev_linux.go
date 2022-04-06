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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ReadRunUdevDataByMajorMinor reads udev data by major minor
func ReadRunUdevDataByMajorMinor(major, minor int) (map[string]string, error) {
	return readRunUdevDataFile(fmt.Sprintf("%v/b%v:%v", runUdevData, major, minor))
}

func readRunUdevDataFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseRunUdevDataFile(file)
}

func parseRunUdevDataFile(r io.Reader) (map[string]string, error) {
	reader := bufio.NewReader(r)
	event := map[string]string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if !strings.HasPrefix(s, "E:") {
			continue
		}

		tokens := strings.SplitN(s, "=", 2)
		key := strings.TrimPrefix(tokens[0], "E:")
		switch len(tokens) {
		case 1:
			event[key] = ""
		case 2:
			event[key] = strings.TrimSpace(tokens[1])
		}
	}
	return event, nil
}

// MapToUdevData maps eventMap to Udev Data
func MapToUdevData(eventMap map[string]string) (*UDevData, error) {
	var err error
	var partition int
	if value, found := eventMap["ID_PART_ENTRY_NUMBER"]; found {
		partition, err = strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
	}

	return &UDevData{
		Partition:        partition,
		WWID:             eventMap["ID_WWN"],
		Model:            eventMap["ID_MODEL"],
		UeventSerial:     eventMap["ID_SERIAL_SHORT"],
		Vendor:           eventMap["ID_VENDOR"],
		DMName:           eventMap["DM_NAME"],
		DMUUID:           eventMap["DM_UUID"],
		MDUUID:           NormalizeUUID(eventMap["MD_UUID"]),
		PTUUID:           eventMap["ID_PART_TABLE_UUID"],
		PTType:           eventMap["ID_PART_TABLE_TYPE"],
		PartUUID:         eventMap["ID_PART_ENTRY_UUID"],
		UeventFSUUID:     eventMap["ID_FS_UUID"],
		FSType:           eventMap["ID_FS_TYPE"],
		UeventSerialLong: eventMap["ID_SERIAL"],
		PCIPath:          eventMap["ID_PATH"],
	}, nil
}
