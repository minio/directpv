//go:build linux

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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultBlockSize = 512

func getDeviceName(majorMinor string) (string, error) {
	filename := fmt.Sprintf("/sys/dev/block/%v/uevent", majorMinor)
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(s, "DEVNAME=") {
			name := strings.TrimSpace(s[8:])
			if name == "" {
				return "", fmt.Errorf("%v contains empty device name for DEVNAME key", filename)
			}
			return name, nil
		}
	}
}

func getHidden(name string) bool {
	// errors ignored since real devices do not have <sys>/hidden
	// borrow idea from 'lsblk'
	// https://github.com/util-linux/util-linux/commit/c8487d854ba5cf5bfcae78d8e5af5587e7622351
	v, _ := readFirstLine("/sys/class/block/" + name + "/hidden")
	return v == "1"
}

func getRemovable(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/removable")
	return s != "" && s != "0", err
}

func getReadOnly(name string) (bool, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/ro")
	return s != "" && s != "0", err
}

func getSize(name string) (uint64, error) {
	s, err := readFirstLine("/sys/class/block/" + name + "/size")
	if err != nil {
		return 0, err
	}
	ui64, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return ui64 * defaultBlockSize, nil
}

func getPartitions(name string) ([]string, error) {
	names, err := readdirnames("/sys/block/"+name, false)
	if err != nil {
		return nil, err
	}

	partitions := []string{}
	for _, n := range names {
		if strings.HasPrefix(n, name) {
			partitions = append(partitions, n)
		}
	}

	return partitions, nil
}

func getHolders(name string) ([]string, error) {
	return readdirnames("/sys/class/block/"+name+"/holders", false)
}

func getDMName(name string) (string, error) {
	return readFirstLine("/sys/class/block/" + name + "/dm/name")
}
