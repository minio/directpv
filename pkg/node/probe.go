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

package node

import (
	"os"
	"strings"

	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

func ProbeDevices() ([]*sys.Device, error) {
	dir, err := os.Open("/run/udev/data")
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var devices []*sys.Device
	for _, name := range names {
		if !strings.HasPrefix(name, "b") {
			continue
		}
		major, minor, err := utils.GetMajorMinorFromStr(strings.TrimPrefix(name, "b"))
		if err != nil {
			klog.V(5).Infof("error while parsing maj:min for file: %s: %v", name, err)
			continue
		}
		devName, err := sys.GetDeviceName(major, minor)
		if err != nil {
			klog.V(5).Infof("error while getting device name for maj:min (%v:%v): %v", major, minor, err)
			continue
		}
		if sys.IsLoopBackDevice("/dev/" + devName) {
			klog.V(5).InfoS("loopback device is ignored while syncing", "DEVNAME", devName)
			continue
		}
		data, err := sys.ReadRunUdevDataByMajorMinor(int(major), int(minor))
		if err != nil {
			klog.V(5).Infof("error while reading udevdata for device %s: %v", devName, err)
			continue
		}
		runUdevData, err := sys.MapToUdevData(data)
		if err != nil {
			klog.V(5).Infof("error while mapping udevdata for device %s: %v", devName, err)
			continue
		}
		device := &sys.Device{
			Name:              devName,
			Major:             int(major),
			Minor:             int(minor),
			Virtual:           strings.Contains(devName, "virtual"),
			Partition:         runUdevData.Partition,
			WWID:              runUdevData.WWID,
			WWIDWithExtension: runUdevData.WWIDWithExtension,
			Model:             runUdevData.Model,
			UeventSerial:      runUdevData.UeventSerial,
			Vendor:            runUdevData.Vendor,
			DMName:            runUdevData.DMName,
			DMUUID:            runUdevData.DMUUID,
			MDUUID:            runUdevData.MDUUID,
			PTUUID:            runUdevData.PTUUID,
			PTType:            runUdevData.PTType,
			PartUUID:          runUdevData.PartUUID,
			UeventFSUUID:      runUdevData.UeventFSUUID,
			FSType:            runUdevData.FSType,
			PCIPath:           runUdevData.PCIPath,
			SerialLong:        runUdevData.UeventSerialLong,
			UDevData:          data,
		}
		// Probe from /sys/
		if err := device.ProbeSysInfo(); err != nil {
			klog.V(5).Infof("error while probing sys info for device %s: %v", devName, err)
			continue
		}
		// Probe from /proc/1/mountinfo
		if err := device.ProbeMountInfo(); err != nil {
			klog.V(5).Infof("error while probing dev info for device %s: %v", devName, err)
			continue
		}
		// Opens the device `/dev/` to probe
		if err := device.ProbeDevInfo(); err != nil {
			klog.V(5).Infof("error while validating device %s: %v", devName, err)
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}
