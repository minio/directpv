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
	"errors"
	"os"

	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
)

func newDevice(mountMap map[string][]string, cdroms, swaps map[string]struct{}, name, majorMinor string, udevData map[string]string) (device *Device, err error) {
	_, swapFound := swaps[utils.AddDevPrefix(name)]
	_, cdromFound := cdroms[name]

	device = &Device{
		Name:        name,
		MajorMinor:  majorMinor,
		MountPoints: mountMap[utils.AddDevPrefix(name)],
		SwapOn:      swapFound,
		CDROM:       cdromFound,
		UDevData:    udevData,
	}

	if device.Size, err = getSize(name); err != nil {
		return nil, err
	}

	device.Hidden = getHidden(name)

	if device.Removable, err = getRemovable(name); err != nil {
		return nil, err
	}

	if device.ReadOnly, err = getReadOnly(name); err != nil {
		return nil, err
	}

	if device.Holders, err = getHolders(name); err != nil {
		return nil, err
	}

	if partitions, err := getPartitions(name); err != nil {
		return nil, err
	} else {
		device.Partitioned = len(partitions) != 0
	}

	if device.DMName, err = getDMName(name); err != nil {
		return nil, err
	}

	return device, nil
}

func probe() (devices []Device, err error) {
	deviceMap, udevDataMap, err := probeFromUdev()
	if err != nil {
		return nil, err
	}

	_, mountMap, err := sys.GetMounts()
	if err != nil {
		return nil, err
	}

	cdroms, err := getCDROMs()
	if err != nil {
		return nil, err
	}

	swaps, err := getSwaps()
	if err != nil {
		return nil, err
	}

	for name, udevData := range udevDataMap {
		device, err := newDevice(mountMap, cdroms, swaps, name, deviceMap[name], udevData)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}

	return devices, nil
}

func probeDevices(majorMinor ...string) (devices []Device, err error) {
	_, mountMap, err := sys.GetMounts()
	if err != nil {
		return nil, err
	}

	cdroms, err := getCDROMs()
	if err != nil {
		return nil, err
	}

	swaps, err := getSwaps()
	if err != nil {
		return nil, err
	}

	for i := range majorMinor {
		udevData, err := readUdevData(majorMinor[i])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		name, err := getDeviceName(majorMinor[i])
		if err != nil {
			return nil, err
		}

		device, err := newDevice(mountMap, cdroms, swaps, name, majorMinor[i], udevData)
		if err != nil {
			return nil, err
		}

		devices = append(devices, *device)
	}

	return devices, nil
}
