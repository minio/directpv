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

package device

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getDeviceByFSUUID(fsuuid string) (device string, err error) {
	if device, err = filepath.EvalSymlinks("/dev/disk/by-uuid/" + fsuuid); err == nil {
		device = filepath.ToSlash(device)
	}
	return
}

func getDeviceName(major, minor uint32) (string, error) {
	filename := fmt.Sprintf("/sys/dev/block/%v:%v/uevent", major, minor)
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

		if !strings.HasPrefix(s, "DEVNAME=") {
			continue
		}

		switch tokens := strings.SplitN(s, "=", 2); len(tokens) {
		case 2:
			return strings.TrimSpace(tokens[1]), nil
		default:
			return "", fmt.Errorf("filename %v contains invalid DEVNAME value", filename)
		}
	}
}
