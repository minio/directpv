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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/glog"
	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/dev"
	"github.com/minio/direct-csi/pkg/utils"
	simd "github.com/minio/sha256-simd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	k8sExec "k8s.io/utils/exec"
	"k8s.io/utils/mount"
)

func FindDrives(ctx context.Context, nodeID string, procfs string) ([]direct_csi.DirectCSIDrive, error) {

	var drives []direct_csi.DirectCSIDrive

	blockDevices, err := dev.FindDevices(ctx)
	if err != nil {
		return drives, err
	}

	for _, blockDevice := range blockDevices {

		if err := blockDevice.Init(ctx, procfs); err != nil {
			glog.V(5).Infof("Failed to initialize the block device: (%s) err: (%s)", blockDevice.Devname, err.Error())
			continue
		}

		partitions := blockDevice.GetPartitions()
		if len(partitions) > 0 {
			for _, partition := range partitions {
				drive, dErr := makePartitionDrive(nodeID, partition, blockDevice.Devname)
				if dErr != nil {
					glog.V(5).Infof("Failed to initialize the partition of block device: (%s) err: (%s)", blockDevice.Devname, dErr.Error())
					continue
				}
				drives = append(drives, *drive)
			}
			continue
		}

		drive, err := makeRootDrive(nodeID, blockDevice)
		if err != nil {
			return nil, err
		}
		drives = append(drives, *drive)

	}

	return drives, nil
}

func makePartitionDrive(nodeID string, partition dev.Partition, rootPartition string) (*direct_csi.DirectCSIDrive, error) {

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

	return &direct_csi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: makeName(nodeID, partition.Path),
		},
		Status: direct_csi.DirectCSIDriveStatus{
			DriveStatus:       getDriveStatus(fs),
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

func makeRootDrive(nodeID string, blockDevice *dev.BlockDevice) (*direct_csi.DirectCSIDrive, error) {

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

	return &direct_csi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: makeName(nodeID, blockDevice.Path),
		},
		Status: direct_csi.DirectCSIDriveStatus{
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

func getDriveStatus(filesystem string) direct_csi.DriveStatus {
	if filesystem == "" {
		return direct_csi.Unformatted
	} else {
		return direct_csi.New
	}
}

// MountDevice - Utility to mount a device in the given mountpoint
func MountDevice(devicePath, mountPoint, fsType string, options []string) error {
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}
	if err := mount.New("").Mount(devicePath, mountPoint, fsType, options); err != nil {
		glog.V(5).Info(err)
		return err
	}
	return nil
}

// FormatDevice - Formats the given device
func FormatDevice(ctx context.Context, source, fsType string, force bool) error {
	args := []string{source}
	forceFlag := "-F"
	if fsType == "xfs" {
		forceFlag = "-f"
	}
	if force {
		args = []string{
			forceFlag, // Force flag
			source,
		}
	}
	glog.V(5).Infof("args: %v", args)
	output, err := exec.CommandContext(ctx, "mkfs."+fsType, args...).CombinedOutput()
	if err != nil {
		glog.V(5).Infof("Failed to format the device: err: (%s) output: (%s)", err.Error(), string(output))
	}
	return err
}

// UnmountIfMounted - Idempotent function to unmount a target
func UnmountIfMounted(mountPoint string) error {
	shouldUmount := false
	mountPoints, mntErr := mount.New("").List()
	if mntErr != nil {
		return mntErr
	}
	for _, mp := range mountPoints {
		abPath, _ := filepath.Abs(mp.Path)
		if mountPoint == abPath {
			shouldUmount = true
			break
		}
	}
	if shouldUmount {
		if mErr := mount.New("").Unmount(mountPoint); mErr != nil {
			return mErr
		}
	}
	return nil
}

// UnmountAllMountRefs - Unmount all mount refs. To avoid later mounts to overlap earlier mounts.
func UnmountAllMountRefs(mountPoint string) error {
	mountRefs, err := mount.New("").GetMountRefs(mountPoint)
	if err != nil {
		return err
	}
	for _, mp := range mountRefs {
		abPath, _ := filepath.Abs(mp)
		if mErr := mount.New("").Unmount(abPath); mErr != nil {
			return mErr
		}
	}
	return nil
}

// GetLatestStatus gets the latest condition by time
func GetLatestStatus(statusXs []metav1.Condition) metav1.Condition {
	// Sort the drives by LastTransitionTime [Descending]
	sort.SliceStable(statusXs, func(i, j int) bool {
		return (&statusXs[j].LastTransitionTime).Before(&statusXs[i].LastTransitionTime)
	})
	return statusXs[0]
}

// GetDiskFS - To get the filesystem of a block device
func GetDiskFS(devicePath string) (string, error) {
	diskMounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: k8sExec.New()}
	// Internally uses 'blkid' to see if the given disk is unformatted
	fs, err := diskMounter.GetDiskFormat(devicePath)
	if err != nil {
		glog.V(5).Infof("Error while reading the disk format: (%s)", err.Error())
	}
	return fs, err
}

// AddDriveFinalizersWithConflictRetry - appends a finalizer to the csidrive's finalizers list
func AddDriveFinalizersWithConflictRetry(ctx context.Context, csiDriveName string, finalizers []string) error {
	directCSIClient := utils.GetDirectCSIClient()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, csiDriveName, metav1.GetOptions{})
		if dErr != nil {
			return dErr
		}
		copiedDrive := csiDrive.DeepCopy()
		for _, finalizer := range finalizers {
			copiedDrive.ObjectMeta.SetFinalizers(utils.AddFinalizer(&copiedDrive.ObjectMeta, finalizer))
		}
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{})
		return err
	}); err != nil {
		glog.V(5).Infof("Error while adding finalizers to csidrive: (%s)", err.Error())
		return err
	}
	return nil
}

// RemoveDriveFinalizerWithConflictRetry - removes a finalizer from the csidrive's finalizers list
func RemoveDriveFinalizerWithConflictRetry(ctx context.Context, csiDriveName string, finalizer string) error {
	directCSIClient := utils.GetDirectCSIClient()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, csiDriveName, metav1.GetOptions{})
		if dErr != nil {
			return dErr
		}
		copiedDrive := csiDrive.DeepCopy()
		copiedDrive.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&copiedDrive.ObjectMeta, finalizer))
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{})
		return err
	}); err != nil {
		glog.V(5).Infof("Error while adding finalizers to csidrive: (%s)", err.Error())
		return err
	}
	return nil
}
