//go:build !linux

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
	"fmt"
	"runtime"
)

func getDeviceMajorMinor(device string) (major, minor uint32, err error) {
	return 0, 0, fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func probeDevices() (devices map[string]*Device, err error) {
	return nil, fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func getDeviceName(major, minor uint32) (string, error) {
	return "", fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

// CreateDevice creates new device from udev data and probes dev and sys
// to fill the remaining device information.
func CreateDevice(udevData *UDevData) (device *Device, err error) {
	return nil, fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}
