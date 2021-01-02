// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/dev"

	"github.com/golang/glog"
	simd "github.com/minio/sha256-simd"
)

func findDrives(ctx context.Context, nodeID string, procfs string) ([]directv1alpha1.DirectCSIDrive, error) {
	drives := []directv1alpha1.DirectCSIDrive{}
	devs, err := dev.FindDevices(ctx)
	if err != nil {
		return drives, err
	}

	for _, d := range devs {
		partitions := d.GetPartitions()
		if len(partitions) > 0 {
			for _, partition := range partitions {
				drive, err := makePartitionDrive(nodeID, partition, d.Devname)
				if err != nil {
					glog.Errorf("Error discovering parition %s: %v", d.Devname, err)
					continue
				}
				drives = append(drives, *drive)
			}
			continue
		}

		drive, err := makeRootDrive(nodeID, d)
		if err != nil {
			return nil, err
		}
		drives = append(drives, *drive)
	}

	return drives, nil
}

func makePartitionDrive(nodeID string, partition dev.Partition, rootPartition string) (*directv1alpha1.DirectCSIDrive, error) {
	var fs string
	if partition.FSInfo != nil {
		fs = string(partition.FSInfo.FSType)
	}

	var freeCapacity, totalCapacity int64
	if partition.FSInfo != nil {
		freeCapacity = int64(partition.FSInfo.FreeCapacity)
		totalCapacity = int64(partition.FSInfo.TotalCapacity)
	}

	var mountOptions []string
	var mountPoint string
	var mounts []dev.Mount
	if partition.FSInfo != nil {
		mounts = partition.FSInfo.Mounts
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].MountPoint
		}
	}

	driveStatus := getDriveStatus(fs)
	_, ok := dev.SystemPartitionTypes[partition.TypeUUID]
	if ok || mountPoint == "/" {
		// system partition
		driveStatus = directv1alpha1.Unavailable
	}

	return &directv1alpha1.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: makeName(nodeID, partition.Path),
		},
		Status: directv1alpha1.DirectCSIDriveStatus{
			DriveStatus:       driveStatus,
			Filesystem:        fs,
			FreeCapacity:      freeCapacity,
			LogicalBlockSize:  int64(partition.LogicalBlockSize),
			ModelNumber:       "", // Fix Me
			MountOptions:      mountOptions,
			Mountpoint:        mountPoint,
			NodeName:          nodeID,
			PartitionNum:      int(partition.PartitionNum),
			Path:              partition.Path,
			PhysicalBlockSize: int64(partition.PhysicalBlockSize),
			RootPartition:     rootPartition,
			SerialNumber:      "", // Fix me
			TotalCapacity:     totalCapacity,
		},
	}, nil
}

func makeRootDrive(nodeID string, blockDevice dev.BlockDevice) (*directv1alpha1.DirectCSIDrive, error) {
	var fs string
	if blockDevice.FSInfo != nil {
		fs = string(blockDevice.FSInfo.FSType)
	}

	var freeCapacity, totalCapacity int64
	if blockDevice.FSInfo != nil {
		freeCapacity = int64(blockDevice.FSInfo.FreeCapacity)
		totalCapacity = int64(blockDevice.FSInfo.TotalCapacity)
	}

	var mountOptions []string
	var mountPoint string
	var mounts []dev.Mount

	if blockDevice.FSInfo != nil {
		mounts = blockDevice.FSInfo.Mounts
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].MountPoint
		}
	}

	return &directv1alpha1.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: makeName(nodeID, blockDevice.Path),
		},
		Status: directv1alpha1.DirectCSIDriveStatus{
			DriveStatus:       getDriveStatus(fs),
			Filesystem:        fs,
			FreeCapacity:      freeCapacity,
			LogicalBlockSize:  int64(blockDevice.LogicalBlockSize),
			ModelNumber:       "", // Fix Me
			MountOptions:      mountOptions,
			Mountpoint:        mountPoint,
			NodeName:          nodeID,
			PartitionNum:      int(0),
			Path:              blockDevice.Path,
			PhysicalBlockSize: int64(blockDevice.PhysicalBlockSize),
			RootPartition:     blockDevice.Devname,
			SerialNumber:      "", // Fix me
			TotalCapacity:     totalCapacity,
		},
	}, nil
}

func makeName(nodeID, path string) string {
	driveName := strings.Join([]string{nodeID, path}, "-")
	return fmt.Sprintf("%x", simd.Sum256([]byte(driveName)))
}

func getDriveStatus(filesystem string) directv1alpha1.DriveStatus {
	if filesystem == "" {
		return directv1alpha1.Unformatted
	} else {
		return directv1alpha1.New
	}
}
