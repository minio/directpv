// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/sys/gpt"
	x "github.com/minio/direct-csi/pkg/sys/xfs"
	"github.com/minio/direct-csi/pkg/utils"
	kerr "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/google/uuid"
)

const (
	loopBackDeviceCount = 4
)

var (
	ErrNoLinkFound = errors.New("No link found for the device")
	ErrInvalidLink = errors.New("Invalid device link found")
)

func findDrives(ctx context.Context, nodeID string, procfs string, loopBackOnly bool) ([]directcsi.DirectCSIDrive, error) {
	drives := []directcsi.DirectCSIDrive{}

	if loopBackOnly {
		// Flush the existing loopback setups
		if err := sys.FlushLoopBackReservations(); err != nil {
			return drives, err
		}
		// Reserve loopbacks
		if err := sys.ReserveLoopbackDevices(loopBackDeviceCount); err != nil {
			return drives, err
		}
	}

	devs, err := sys.FindDevices(ctx, loopBackOnly)
	if err != nil {
		return drives, err
	}

	for _, d := range devs {
		partitions := d.GetPartitions()
		if len(partitions) > 0 {
			for _, partition := range partitions {
				drive, err := makePartitionDrive(ctx, nodeID, partition, d.Devname, d.DeviceError)
				if err != nil {
					glog.Errorf("Error discovering parition %s: %v", d.Devname, err)
					continue
				}
				drives = append(drives, *drive)
			}
			continue
		}

		drive, err := makeRootDrive(ctx, nodeID, d)
		if err != nil {
			return nil, err
		}
		drives = append(drives, *drive)
	}

	return drives, nil
}

func makePartitionDrive(ctx context.Context, nodeID string, partition sys.Partition, rootPartition string, blockErr error) (*directcsi.DirectCSIDrive, error) {
	var fs string
	if partition.FSInfo != nil {
		fs = string(partition.FSInfo.FSType)
	}

	var allocatedCapacity, freeCapacity, totalCapacity int64
	if partition.FSInfo != nil {
		freeCapacity = int64(partition.FSInfo.FreeCapacity)
		totalCapacity = int64(partition.FSInfo.TotalCapacity)
		allocatedCapacity = totalCapacity - freeCapacity
	}

	var mountOptions []string
	var mountPoint string
	var mounts []sys.MountInfo
	var driveStatus directcsi.DriveStatus

	driveStatus = directcsi.DriveStatusAvailable
	if partition.FSInfo != nil {
		mounts = partition.FSInfo.Mounts
		for _, m := range mounts {
			if m.Mountpoint == "/" {
				driveStatus = directcsi.DriveStatusUnavailable
			}
		}
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].Mountpoint
		}
	}
	_, ok := gpt.SystemPartitionTypes[partition.TypeUUID]
	if ok || blockErr != nil {
		driveStatus = directcsi.DriveStatusUnavailable
	}

	blockInitializationStatus := metav1.ConditionTrue
	if blockErr != nil {
		blockInitializationStatus = metav1.ConditionFalse
	}

	mounted := metav1.ConditionFalse
	formatted := metav1.ConditionFalse
	if fs != "" {
		formatted = metav1.ConditionTrue
	}
	if mountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	driveName, err := makeDevice(ctx, partition.Path, partition.DriveInfo)
	if err != nil {
		return nil, err
	}

	return &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: driveName,
		},
		Status: directcsi.DirectCSIDriveStatus{
			DriveStatus:       driveStatus,
			Filesystem:        fs,
			FreeCapacity:      freeCapacity,
			AllocatedCapacity: allocatedCapacity,
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
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             mounted,
					Message:            mountPoint,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             formatted,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:   string(directcsi.DirectCSIDriveConditionInitialized),
					Status: blockInitializationStatus,
					Message: func() string {
						if blockErr == nil {
							return ""
						}
						return blockErr.Error()
					}(),
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}, nil
}

func makeRootDrive(ctx context.Context, nodeID string, blockDevice sys.BlockDevice) (*directcsi.DirectCSIDrive, error) {
	var fs string
	if blockDevice.FSInfo != nil {
		fs = string(blockDevice.FSInfo.FSType)
	}

	var freeCapacity, totalCapacity, allocatedCapacity int64
	if blockDevice.FSInfo != nil {
		freeCapacity = int64(blockDevice.FSInfo.FreeCapacity)
		totalCapacity = int64(blockDevice.FSInfo.TotalCapacity)
		allocatedCapacity = totalCapacity - freeCapacity
	}

	var mountOptions []string
	var mountPoint string
	var mounts []sys.MountInfo
	var driveStatus directcsi.DriveStatus

	driveStatus = directcsi.DriveStatusAvailable
	if blockDevice.FSInfo != nil {
		mounts = blockDevice.FSInfo.Mounts
		for _, m := range mounts {
			if m.Mountpoint == "/" {
				driveStatus = directcsi.DriveStatusUnavailable
			}
		}
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].Mountpoint
		}
	}

	blockInitializationStatus := metav1.ConditionTrue
	if blockDevice.DeviceError != nil {
		driveStatus = directcsi.DriveStatusUnavailable
		blockInitializationStatus = metav1.ConditionFalse
	}

	mounted := metav1.ConditionFalse
	formatted := metav1.ConditionFalse
	if fs != "" {
		formatted = metav1.ConditionTrue
	}
	if mountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	driveName, err := makeDevice(ctx, blockDevice.Path, blockDevice.DriveInfo)
	if err != nil {
		return nil, err
	}

	return &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: driveName,
		},
		Status: directcsi.DirectCSIDriveStatus{
			DriveStatus:       driveStatus,
			Filesystem:        fs,
			FreeCapacity:      freeCapacity,
			AllocatedCapacity: allocatedCapacity,
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
			Conditions: []metav1.Condition{
				{
					Type:               string(directcsi.DirectCSIDriveConditionOwned),
					Status:             metav1.ConditionFalse,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionMounted),
					Status:             mounted,
					Message:            mountPoint,
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionFormatted),
					Status:             formatted,
					Message:            "xfs",
					Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               string(directcsi.DirectCSIDriveConditionInitialized),
					Status:             blockInitializationStatus,
					Message:            blockDevice.Error(),
					Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}, nil
}

func makeDevice(ctx context.Context, directCSIDrivePath string, driveInfo *sys.DriveInfo) (string, error) {
	driveName, err := getDeviceName(ctx, directCSIDrivePath, driveInfo)
	if err != nil {
		if err != ErrNoLinkFound && err != ErrInvalidLink {
			return "", err
		}
	}
	directCSIDevicePath := sys.GetDirectCSIPath(driveName)
	if err := sys.MakeBlockFile(directCSIDevicePath, driveInfo.Major, driveInfo.Minor); err != nil {
		return "", err
	}
	if err := sys.MakeLinkFile(directCSIDevicePath, getLinkPath(directCSIDrivePath)); err != nil {
		return "", err
	}

	return driveName, nil
}

func getMajorMinor(devicePath string) (uint32, uint32, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Stat(devicePath, &stat); err != nil {
		return uint32(0), uint32(0), err
	}
	dev := stat.Rdev
	return uint32(unix.Major(dev)), uint32(unix.Minor(dev)), nil
}

func deleteNonXFSDrive(ctx context.Context, driveName string) error {
	directClient := utils.GetDirectCSIClient()
	dClient := directClient.DirectCSIDrives()

	driveObj, dErr := dClient.Get(ctx, driveName, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
	})
	if dErr != nil {
		if kerr.IsNotFound(dErr) {
			// This can happen if the node restarted during the drive discovery process and the drive order is changed.
			return nil
		}
		return dErr
	}

	if driveObj.Status.Filesystem != string(sys.FSTypeXFS) {
		// Remove the device path if it is a no-FS or no-XFS drive.
		// Such paths cannot be re-used and will be newly created when the drive order changes
		directCSIDevicePath := sys.GetDirectCSIPath(driveObj.Name)
		if err := os.Remove(directCSIDevicePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := dClient.Delete(ctx, driveName, metav1.DeleteOptions{}); err != nil {
			if !kerr.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func getLinkPath(path string) string {
	return filepath.Join(sys.DirectCSILinksDir, filepath.Base(path))
}

func getBlockDevicePath(directCSIDevicePath string, major, minor uint32) (string, error) {
	deleteLink := func(linkPath string) error {
		if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	linkPath := getLinkPath(directCSIDevicePath)
	if _, err := os.Stat(linkPath); err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoLinkFound
		}
		return "", err
	}
	lnk, lErr := os.Readlink(linkPath)
	if lErr != nil {
		if os.IsNotExist(lErr) {
			// Delete the stale/invalid link
			if err := deleteLink(linkPath); err != nil {
				return "", err
			}
			return "", ErrInvalidLink
		}
		return "", lErr
	}

	return lnk, nil
}

func cleanUpDeviceEntries(blockDevicePath, deviceLinkPath string) error {
	if err := os.Remove(blockDevicePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(deviceLinkPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func getDeviceName(ctx context.Context, directCSIDrivePath string, driveInfo *sys.DriveInfo) (string, error) {
	fsInfo := driveInfo.FSInfo

	getDriveName := func(fsInfo *sys.FSInfo) string {
		switch fsInfo.FSType {
		case x.FSTypeXFS:
			return string(fsInfo.UUID)
		default:
			return uuid.New().String()
		}
	}

	blkDevicePath, err := getBlockDevicePath(directCSIDrivePath, driveInfo.Major, driveInfo.Minor)
	if err != nil {
		return getDriveName(fsInfo), err
	}

	blkMajor, blkMinor, err := getMajorMinor(blkDevicePath)
	if err != nil {
		if err := cleanUpDeviceEntries(blkDevicePath, getLinkPath(directCSIDrivePath)); err != nil {
			return "", err
		}
		return getDriveName(fsInfo), ErrInvalidLink
	}

	if blkMajor == driveInfo.Major && blkMinor == driveInfo.Minor {
		return filepath.Base(blkDevicePath), nil
	}

	// Remove the corresponding device link as the drive order has changed
	if err := os.Remove(getLinkPath(directCSIDrivePath)); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	// Delete the drive object if it is not a XFS drive.
	// Note: The no-FS and no-XFS drives and its corresponding device paths will
	// be recreated if the order is changed.
	if err := deleteNonXFSDrive(ctx, filepath.Base(blkDevicePath)); err != nil {
		return getDriveName(fsInfo), err
	}

	return getDriveName(fsInfo), nil
}
