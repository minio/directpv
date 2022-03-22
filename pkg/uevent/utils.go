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
	"path/filepath"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/sys"
	"k8s.io/klog/v2"
)

func getRootBlockPath(devName string) string {
	switch {
	case strings.HasPrefix(devName, sys.HostDevRoot):
		return devName
	case strings.Contains(devName, sys.DirectCSIDevRoot):
		return getRootBlockPath(filepath.Base(devName))
	default:
		name := strings.ReplaceAll(
			strings.Replace(devName, sys.DirectCSIPartitionInfix, "", 1),
			sys.DirectCSIPartitionInfix,
			sys.HostPartitionInfix,
		)
		return filepath.Join(sys.HostDevRoot, name)
	}
}

// ValidateDevice validates the device with drive
func ValidateDevice(device *sys.Device, directCSIDrive *directcsi.DirectCSIDrive) bool {

	if directCSIDrive.Status.Path != device.DevPath() {
		klog.V(3).Infof("[%s] path mismatch: %v -> %v", directCSIDrive.Status.Path, device.DevPath())
		return false
	}
	if directCSIDrive.Status.MajorNumber != uint32(device.Major) {
		klog.V(3).Infof("[%s] major number mismatch: %v -> %v", device.Name, directCSIDrive.Status.MajorNumber, device.Major)
		return false
	}
	if directCSIDrive.Status.MinorNumber != uint32(device.Minor) {
		klog.V(3).Infof("[%s] minor number mismatch: %v -> %v", device.Name, directCSIDrive.Status.MinorNumber, device.Minor)
		return false
	}
	if directCSIDrive.Status.PartitionNum != device.Partition {
		klog.V(3).Infof("[%s] partitionnum mismatch: %v -> %v", device.Name, directCSIDrive.Status.PartitionNum, device.Partition)
		return false
	}
	if directCSIDrive.Status.WWID != device.WWID {
		klog.V(3).Infof("[%s] wwid msmatch: %v -> %v", device.Name, directCSIDrive.Status.WWID, device.WWID)
		return false
	}
	if directCSIDrive.Status.ModelNumber != device.Model {
		klog.V(3).Infof("[%s] modelnumber mismatch: %v -> %v", device.Name, directCSIDrive.Status.ModelNumber, device.Model)
		return false
	}
	if directCSIDrive.Status.UeventSerial != device.UeventSerial {
		klog.V(3).Infof("[%s] ueventserial mismatch: %v -> %v", device.Name, directCSIDrive.Status.UeventSerial, device.UeventSerial)
		return false
	}
	if directCSIDrive.Status.Vendor != device.Vendor {
		klog.V(3).Infof("[%s] vendor mismatch: %v -> %v", device.Name, directCSIDrive.Status.Vendor, device.Vendor)
		return false
	}
	if directCSIDrive.Status.DMName != device.DMName {
		klog.V(3).Infof("[%s] dmname mismatch: %v -> %v", device.Name, directCSIDrive.Status.DMName, device.DMName)
		return false
	}
	if directCSIDrive.Status.DMUUID != device.DMUUID {
		klog.V(3).Infof("[%s] dmuuid mismatch: %v -> %v", device.Name, directCSIDrive.Status.DMUUID, device.DMUUID)
		return false
	}
	if directCSIDrive.Status.MDUUID != device.MDUUID {
		klog.V(3).Infof("[%s] MDUUID mismatch: %v -> %v", device.Name, directCSIDrive.Status.MDUUID, device.MDUUID)
		return false
	}
	if directCSIDrive.Status.PartTableUUID != device.PTUUID {
		klog.V(3).Infof("[%s] PartTableUUID mismatch: %v -> %v", device.Name, directCSIDrive.Status.PartTableUUID, device.PTUUID)
		return false
	}
	if directCSIDrive.Status.PartTableType != device.PTType {
		klog.V(3).Infof("[%s] PartTableType mismatch: %v -> %v", device.Name, directCSIDrive.Status.PartTableType, device.PTType)
		return false
	}
	if directCSIDrive.Status.PartitionUUID != device.PartUUID {
		klog.V(3).Infof("[%s] PartitionUUID mismatch: %v -> %v", device.Name, directCSIDrive.Status.PartitionUUID, device.PartUUID)
		return false
	}
	if directCSIDrive.Status.Filesystem != device.FSType {
		klog.V(3).Infof("[%s] filesystem mismatch: %v -> %v", device.Name, directCSIDrive.Status.Filesystem, device.FSType)
		return false
	}
	if directCSIDrive.Status.UeventFSUUID != device.UeventFSUUID {
		klog.V(3).Infof("[%s] mismatch ueventfsuuid %v - %v", device.Name, directCSIDrive.Status.UeventFSUUID, device.UeventFSUUID)
		return false
	}

	return true
}
