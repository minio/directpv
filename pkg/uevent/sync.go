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

package uevent

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta5"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"k8s.io/klog/v2"
)

var errDuplicateDevice = errors.New("found duplicate devices for drive")

// syncs the directcsidrive states by locally probing the devices
func (l *listener) sync(ctx context.Context) error {
	dir, err := os.Open("/run/udev/data")
	if err != nil {
		return err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
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

	return l.syncDevices(ctx, devices)
}

// Tries to match the directcsidrives by the discovered and probed devices from host
//
// follows group based match, managed (ready|inuse) and non-managed (available|unavailable|released),
// this isolates them and do-not allow matching collision between two groups
//
// Managed directcsidrives (InUse and Ready) :-
// =========================================
// (*) Compares the FS attributes (fstype and fsuuid) in the devices and the directcsidrive.
// (*) Matched directcsidrive will be updated.
// (*) Unmatched directcsidrive will be tagged as "lost".
// (*) Too many device matches will be logged as an error.
// (*) unmatched devices will be used for the next directcsidrive match in the iteration.
//
// Non-Managed directcsidrives (Available, Unavailable, Released) :-
// ==============================================================
// (*) Tries to match the directcsidrive by the following attributes if available on the directcsidrive
//
//	(1)  PartitionNumber
//	(2)  WWID
//	(3)  Serial
//	(4)  SerialLong
//	(5)  DMUUID
//	(6)  MDUUID
//	(7)  ModelNumber
//	(8)  Vendor
//	(9)  PartitionUUID
//	(10) UeventFSUUID
//	(11) FilesystemUUID (only matcher for v1.4.6 drives)
//
// (*) Matched directcsidrive will be updated.
// (*) Unmatched directcsidrive will be considered as un-identified directcsidrive. These
//
//	directcsidrive could be non-upgraded or empty drives (virtual) with no persistent attributes in them
//
// (*) Too many device matches will be logged as an error.
// (*) Unmatched devices will be used for the next directcsidrive match in the iteration.
//
// Un-identified directcsidrives (unmatched non-managed directcsidrives) :-
// =====================================================================
// (*) Tries to match the directcsidrive by the following attributes on the directcsidrive
//
//	(1) drive path
//	(2) major and minor number
//	(3) PCIPath (if present)
//
// (*) Matched directcsidrive will be updated.
// (*) Unmatched directcsidrive will be deleted
// (*) Too many device matches will be logged as an error
// (*) Unmatched devices will be created
func (l *listener) syncDevices(ctx context.Context, devices []*sys.Device) error {
	managedDrives, nonManagedDrives, err := l.indexer.listDrives()
	if err != nil {
		return err
	}
	drives := append(managedDrives, nonManagedDrives...)

	var lostDrives, unidentifiedDrives []*directcsi.DirectCSIDrive
	var matchedDevices, unmatchedDevices []*sys.Device
	for _, drive := range drives {
		var managedDrive bool
		if utils.IsManagedDrive(drive) {
			matchedDevices, unmatchedDevices = getMatchedDevicesForManagedDrive(drive, devices)
			managedDrive = true
		} else {
			matchedDevices, unmatchedDevices = getMatchedDevicesForNonManagedDrive(drive, devices)
		}
		switch len(matchedDevices) {
		case 0:
			if !managedDrive {
				unidentifiedDrives = append(unidentifiedDrives, drive)
			} else {
				lostDrives = append(lostDrives, drive)
			}
		case 1:
			klog.V(5).Infof("matched device: %s for drive %s", matchedDevices[0].Name, drive.Name)
			if err := l.processMatchedDrive(ctx, matchedDevices[0], drive); err != nil {
				klog.V(3).Infof("error while processing update for device %s: %v", drive.Status.Path, err)
			}
		default:
			klog.ErrorS(errDuplicateDevice, "drive: ", drive.Name, " devices: ", getDeviceNames(matchedDevices))
		}
		devices = unmatchedDevices
	}

	for _, drive := range unidentifiedDrives {
		matchedDevices, unmatchedDevices = getMatchedDevicesForUnidentifiedDrive(drive, devices)
		switch len(matchedDevices) {
		case 0:
			lostDrives = append(lostDrives, drive)
		case 1:
			klog.V(5).Infof("matched device: %s for drive %s", matchedDevices[0].Name, drive.Name)
			// unset requested format while matching unidentified drive
			if drive.Spec.RequestedFormat != nil && matchedDevices[0].FSUUID != drive.Name && matchedDevices[0].FirstMountPoint != path.Join(sys.MountRoot, drive.Name) {
				drive.Spec.RequestedFormat = nil
			}
			if err := l.processMatchedDrive(ctx, matchedDevices[0], drive); err != nil {
				klog.V(3).Infof("error while processing update for device %s: %v", drive.Status.Path, err)
			}
		default:
			klog.ErrorS(errDuplicateDevice, "drive: ", drive.Name, " devices: ", getDeviceNames(matchedDevices))
		}
		devices = unmatchedDevices
	}

	for _, drive := range lostDrives {
		if err := l.handler.Remove(ctx, drive); err != nil {
			klog.V(3).Infof("error while removing drive %s: %v", drive.Name, err)
		}
	}

	for _, device := range devices {
		if err := l.handler.Add(ctx, device); err != nil {
			klog.V(3).Infof("error while adding device %s: %v", device.Name, err)
		}
	}

	return nil
}

func (l *listener) processMatchedDrive(ctx context.Context, device *sys.Device, drive *directcsi.DirectCSIDrive) error {
	matchResult := noChange
	if isChanged(device, drive) || IsFormatRequested(drive) {
		matchResult = changed
	}
	return l.processUpdate(ctx, matchResult, device, drive)
}
