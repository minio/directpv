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
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/minio/directpv/pkg/consts"
)

var loopDeviceRegexp = regexp.MustCompile("^loop[0-9]*")

func parseUdevData(r io.Reader) (map[string]string, error) {
	reader := bufio.NewReader(r)
	properties := map[string]string{}
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		switch tokens := strings.SplitN(s, "=", 2); len(tokens) {
		case 1:
			properties[tokens[0]] = ""
		case 2:
			properties[tokens[0]] = strings.TrimSpace(tokens[1])
		}
	}
	return properties, nil
}

func readUdevDataFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseUdevData(file)
}

func readUdevData(majorMinor string) (map[string]string, error) {
	file, err := os.Open(path.Join(consts.UdevDataDir, "b"+majorMinor))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseUdevData(file)
}

func probeFromUdev() (deviceMap map[string]string, udevDataMap map[string]map[string]string, err error) {
	dir, err := os.Open(consts.UdevDataDir)
	if err != nil {
		return nil, nil, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, nil, err
	}

	deviceMap = map[string]string{}
	udevDataMap = map[string]map[string]string{}
	for _, name := range names {
		if !strings.HasPrefix(name, "b") {
			continue
		}

		majorMinor := name[1:]

		deviceName, err := getDeviceName(majorMinor)
		if err != nil {
			return nil, nil, err
		}

		if loopDeviceRegexp.MatchString(deviceName) {
			continue
		}

		deviceMap[deviceName] = majorMinor

		properties, err := readUdevDataFile(path.Join(consts.UdevDataDir, name))
		if err != nil {
			return nil, nil, err
		}

		udevDataMap[deviceName] = properties
	}

	return deviceMap, udevDataMap, nil
}
