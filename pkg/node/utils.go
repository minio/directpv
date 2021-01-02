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
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	kexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/golang/glog"
)

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

// UnmountAllMountRefs - Unmount all mount refs. To avoid later mounts to overlap earlier mounts.
func UnmountAllMountRefs(mountPoint string) error {
	mounter := mount.New("")
	// Get all the mountrefs
	mountRefs, err := mounter.GetMountRefs(mountPoint)
	if err != nil {
		return err
	}
	// If there are no refs, Simply unmount.
	if len(mountRefs) == 0 {
		return utils.UnmountIfMounted(mountPoint)
	}
	// Else, Unmount all the mountrefs first.
	mountPoints, mntErr := mounter.List()
	if mntErr != nil {
		return mntErr
	}
	for _, mRef := range mountRefs {
		abmRef, _ := filepath.Abs(mRef)
		for _, mp := range mountPoints {
			abMp, _ := filepath.Abs(mp.Path)
			if abmRef == abMp {
				if mErr := mounter.Unmount(abmRef); mErr != nil {
					return mErr
				}
			}
		}
	}
	// Finally, unmount the mountpoint
	for _, mp := range mountPoints {
		abPath, _ := filepath.Abs(mp.Path)
		if mountPoint == abPath {
			if mErr := mounter.Unmount(mountPoint); mErr != nil {
				return mErr
			}
			break
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
	diskMounter := &mount.SafeFormatAndMount{Interface: mount.New(""), Exec: kexec.New()}
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

// UpdateDriveStatusOnDiff Updates the drive status fields on diff.
func UpdateDriveStatusOnDiff(newObj directv1alpha1.DirectCSIDrive, existingObj *directv1alpha1.DirectCSIDrive) bool {
	isUpdated := false
	if existingObj.Status.Path != newObj.Status.Path {
		existingObj.Status.Path = newObj.Status.Path
		isUpdated = true
	}
	if existingObj.Status.AllocatedCapacity != newObj.Status.AllocatedCapacity {
		existingObj.Status.AllocatedCapacity = newObj.Status.AllocatedCapacity
		isUpdated = true
	}
	if existingObj.Status.FreeCapacity != newObj.Status.FreeCapacity {
		existingObj.Status.FreeCapacity = newObj.Status.FreeCapacity
		isUpdated = true
	}
	if existingObj.Status.RootPartition != newObj.Status.RootPartition {
		existingObj.Status.RootPartition = newObj.Status.RootPartition
		isUpdated = true
	}
	if existingObj.Status.PartitionNum != newObj.Status.PartitionNum {
		existingObj.Status.PartitionNum = newObj.Status.PartitionNum
		isUpdated = true
	}
	if existingObj.Status.Filesystem != newObj.Status.Filesystem {
		existingObj.Status.Filesystem = newObj.Status.Filesystem
		isUpdated = true
	}
	if existingObj.Status.Mountpoint != newObj.Status.Mountpoint {
		existingObj.Status.Mountpoint = newObj.Status.Mountpoint
		isUpdated = true
	}
	if !reflect.DeepEqual(existingObj.Status.MountOptions, newObj.Status.MountOptions) {
		existingObj.Status.MountOptions = newObj.Status.MountOptions
		isUpdated = true
	}
	if existingObj.Status.NodeName != newObj.Status.NodeName {
		existingObj.Status.NodeName = newObj.Status.NodeName
		isUpdated = true
	}
	if existingObj.Status.DriveStatus != newObj.Status.DriveStatus {
		existingObj.Status.DriveStatus = newObj.Status.DriveStatus
		isUpdated = true
	}
	if existingObj.Status.ModelNumber != newObj.Status.ModelNumber {
		existingObj.Status.ModelNumber = newObj.Status.ModelNumber
		isUpdated = true
	}
	if existingObj.Status.SerialNumber != newObj.Status.SerialNumber {
		existingObj.Status.SerialNumber = newObj.Status.SerialNumber
		isUpdated = true
	}
	if existingObj.Status.TotalCapacity != newObj.Status.TotalCapacity {
		existingObj.Status.TotalCapacity = newObj.Status.TotalCapacity
		isUpdated = true
	}
	if existingObj.Status.PhysicalBlockSize != newObj.Status.PhysicalBlockSize {
		existingObj.Status.PhysicalBlockSize = newObj.Status.PhysicalBlockSize
		isUpdated = true
	}
	if existingObj.Status.LogicalBlockSize != newObj.Status.LogicalBlockSize {
		existingObj.Status.LogicalBlockSize = newObj.Status.LogicalBlockSize
		isUpdated = true
	}
	if !reflect.DeepEqual(existingObj.Status.Topology, newObj.Status.Topology) {
		existingObj.Status.Topology = newObj.Status.Topology
		isUpdated = true
	}
	// Add new status canditates here

	return isUpdated
}
