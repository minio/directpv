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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/sys"
	"k8s.io/klog/v2"
)

func probeDevices() ([]*Device, error) {
	dir, err := os.Open(runUdevData)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var devices []*Device
	for _, name := range names {
		if !strings.HasPrefix(name, "b") {
			continue
		}
		majMinInStr := strings.TrimPrefix(name, "b")
		major, minor, err := getMajorMinorFromStr(majMinInStr)
		if err != nil {
			klog.V(5).Infof("error while parsing maj:min for file: %s: %v", name, err)
			continue
		}
		devName, err := getDeviceName(major, minor)
		if err != nil {
			klog.V(5).Infof("error while getting device name for maj:min (%v:%v): %v", major, minor, err)
			continue
		}
		if isLoopBackDevice("/dev/" + devName) {
			klog.V(5).InfoS("loopback device is ignored while syncing", "DEVNAME", devName)
			continue
		}
		data, err := ReadRunUdevDataByMajorMinor(majMinInStr)
		if err != nil {
			klog.V(5).Infof("error while reading udevdata for device %s: %v", devName, err)
			continue
		}
		device := &Device{
			Name:       devName,
			MajorMinor: majMinInStr,
			UDevData:   data,
		}
		// Probe from /sys/
		if err := device.probeSysInfo(); err != nil {
			klog.V(5).Infof("error while probing sys info for device %s: %v", devName, err)
			continue
		}
		// Probe from /proc/1/mountinfo
		if err := device.probeMountInfo(); err != nil {
			klog.V(5).Infof("error while probing dev info for device %s: %v", devName, err)
			continue
		}
		// Probe from /proc/
		if err := device.probeProcInfo(); err != nil {
			klog.V(5).Infof("error while probing dev info for device %s: %v", devName, err)
			continue
		}
		// Opens the device `/dev/` to probe XFS
		if err := device.probeDevInfo(); err != nil {
			klog.V(5).Infof("error while validating device %s: %v", devName, err)
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

// ProbeSysInfo probes device information from /sys
func (device *Device) probeSysInfo() (err error) {
	device.Hidden = getHidden(device.Name)
	if device.Removable, err = getRemovable(device.Name); err != nil {
		return err
	}

	if device.ReadOnly, err = getReadOnly(device.Name); err != nil {
		return err
	}

	if device.Size, err = getSize(device.Name); err != nil {
		return err
	}

	// No partitions for hidden devices.
	if !device.Hidden {
		partitionNo, err := device.PartitionNumber()
		if err != nil {
			return err
		}
		if partitionNo <= 0 {
			names, err := getPartitions(device.Name)
			if err != nil {
				return err
			}
			device.Partitioned = len(names) > 0
		}
		device.Holders, err = getHolders(device.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// ProbeMountInfo probes mount information from /proc/1/mountinfo
func (device *Device) probeMountInfo() (err error) {
	_, deviceMap, err := sys.GetMounts()
	if err != nil {
		klog.ErrorS(err, "unable to probe mounts", "device", device.Name)
		return err
	}
	device.MountPoints = deviceMap[device.Path()]
	return nil
}

// ProbeProcInfo probes the device information from /proc
func (device *Device) probeProcInfo() (err error) {
	if !device.Hidden {
		CDROMs, err := getCDROMs()
		if err != nil {
			return err
		}
		if _, found := CDROMs[device.Name]; found {
			device.CDRom = true
		}
		swaps, err := getSwaps()
		if err != nil {
			return err
		}
		if _, found := swaps[device.MajorMinor]; found {
			device.SwapOn = true
		}
	}
	return nil
}

// ProbeDevInfo probes device information from /dev
func (device *Device) probeDevInfo() (err error) {
	// No FS information needed for hidden devices
	if !device.Hidden && !device.CDRom {
		return updateFSInfo(device)
	}
	return nil
}
