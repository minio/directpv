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
	"fmt"
	"os"

	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
)

func newDevice(
	mountInfo *sys.MountInfo,
	cdroms utils.StringSet,
	swaps utils.StringSet,
	name string,
	majorMinor string,
	udevData map[string]string,
) (device *Device, err error) {
	mountPoints := make(utils.StringSet)
	for _, mountEntry := range mountInfo.FilterByMajorMinor(majorMinor).List() {
		mountPoints.Set(mountEntry.MountPoint)
	}

	device = &Device{
		Name:        name,
		MajorMinor:  majorMinor,
		MountPoints: mountPoints.ToSlice(),
		CDROM:       cdroms.Exist(name),
		SwapOn:      swaps.Exist(utils.AddDevPrefix(name)),
		udevData:    udevData,
	}

	if device.Size, err = getSize(name); err != nil {
		return nil, fmt.Errorf("unable to get size; device=%v; err=%w", name, err)
	}

	device.Hidden = getHidden(name)

	if device.Removable, err = getRemovable(name); err != nil {
		return nil, fmt.Errorf("unable to get removable flag; device=%v; err=%w", name, err)
	}

	if device.ReadOnly, err = getReadOnly(name); err != nil {
		return nil, fmt.Errorf("unable to get read-only flag; device=%v; err=%w", name, err)
	}

	if device.Holders, err = getHolders(name); err != nil {
		return nil, fmt.Errorf("unable to get holders; device=%v; %w", name, err)
	}

	partitions, err := getPartitions(name)
	if err != nil {
		return nil, fmt.Errorf("unable to get partition info; device=%v; err=%w", name, err)
	}
	device.Partitioned = len(partitions) != 0

	if device.DMName, err = getDMName(name); err != nil {
		return nil, fmt.Errorf("unable to get DM name; device=%v; err=%w", name, err)
	}

	return device, nil
}

func probe() (devices []Device, err error) {
	deviceMap, udevDataMap, err := probeFromUdev()
	if err != nil {
		return nil, fmt.Errorf("unable to probe from udev; %w", err)
	}

	mountInfo, err := sys.NewMountInfo()
	if err != nil {
		return nil, fmt.Errorf("unable to get mounts; %w", err)
	}

	cdroms, err := getCDROMs()
	if err != nil {
		return nil, fmt.Errorf("unable to get CDROM information; %w", err)
	}

	swaps, err := getSwaps()
	if err != nil {
		return nil, fmt.Errorf("unable to get swap information; %w", err)
	}

	for name, udevData := range udevDataMap {
		device, err := newDevice(mountInfo, cdroms, swaps, name, deviceMap[name], udevData)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}

	return devices, nil
}

func probeDevices(majorMinor ...string) (devices []Device, err error) {
	mountInfo, err := sys.NewMountInfo()
	if err != nil {
		return nil, fmt.Errorf("unable to get mounts; %w", err)
	}

	cdroms, err := getCDROMs()
	if err != nil {
		return nil, fmt.Errorf("unable to get CDROM information; %w", err)
	}

	swaps, err := getSwaps()
	if err != nil {
		return nil, fmt.Errorf("unable to get swap information; %w", err)
	}

	for i := range majorMinor {
		udevData, err := readUdevData(majorMinor[i])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("unable to read udev data; majorminor=%v; err=%w", majorMinor[i], err)
		}

		name, err := getDeviceName(majorMinor[i])
		if err != nil {
			return nil, fmt.Errorf("unable to get device name; majorminor=%v; err=%w", majorMinor[i], err)
		}

		device, err := newDevice(mountInfo, cdroms, swaps, name, majorMinor[i], udevData)
		if err != nil {
			return nil, err
		}

		devices = append(devices, *device)
	}

	return devices, nil
}
