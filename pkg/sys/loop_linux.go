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
	"errors"
	"os"
	"path"
	"strings"

	"gopkg.in/freddierice/go-losetup.v1"
	"k8s.io/klog/v2"
)

const (
	loopFileRoot    = "/var/lib/direct-csi/loop"
	loopDeviceCount = 4
	GiB             = 1024 * 1024 * 1024
)

func createLoopDevices() error {
	var loopFiles []string
	var loopDevices []losetup.Device

	if err := os.Mkdir(loopFileRoot, 0777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	createLoop := func() error {
		file, err := os.CreateTemp(loopFileRoot, "loop.file.")
		if err != nil {
			return err
		}
		file.Close()

		if err = os.Truncate(file.Name(), 1*GiB); err != nil {
			return err
		}

		loopDevice, err := losetup.Attach(file.Name(), 0, false)
		if err != nil {
			return err
		}

		loopDevices = append(loopDevices, loopDevice)
		loopFiles = append(loopFiles, file.Name())
		return nil
	}

	removeLoops := func() {
		for _, loopDevice := range loopDevices {
			if err := loopDevice.Detach(); err != nil {
				klog.Error(err)
			}
		}

		for _, loopFile := range loopFiles {
			os.Remove(loopFile)
		}
	}

	for i := 0; i < loopDeviceCount; i++ {
		if err := createLoop(); err != nil {
			removeLoops()
			return err
		}
	}

	return nil
}

// IsLoopBackDevice checks if the device is a loopback or not
func IsLoopBackDevice(devPath string) bool {
	return strings.HasPrefix(path.Base(devPath), "loop")
}
