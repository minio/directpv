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
	"reflect"
	"sort"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/sys/xfs"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	kexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetLatestStatus gets the latest condition by time
func GetLatestStatus(statusXs []metav1.Condition) metav1.Condition {
	// Sort the drives by LastTransitionTime [Descending]
	sort.SliceStable(statusXs, func(i, j int) bool {
		return (&statusXs[j].LastTransitionTime).Before(&statusXs[i].LastTransitionTime)
	})
	return statusXs[0]
}

// GetDiskFS - To get the filesystem of a block sys.ce
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

// Idempotent function to bind mount a xfs filesystem with limits
func mountVolume(ctx context.Context, src, dest, vID string, size int64, readOnly bool) error {
	if err := sys.SafeMount(src, dest, string(sys.FSTypeXFS),
		func() []sys.MountOption {
			mOpts := []sys.MountOption{
				sys.MountOptionMSBind,
			}
			if readOnly {
				mOpts = append(mOpts, sys.MountOptionMSReadOnly)
			}
			return mOpts
		}(), []string{"prjquota"}); err != nil {
		return err
	}

	if size > 0 {
		xfsQuota := &xfs.XFSQuota{
			Path:      dest,
			ProjectID: vID,
		}
		if err := xfsQuota.SetQuota(ctx, size); err != nil {
			return status.Errorf(codes.Internal, "Error while setting xfs limits: %v", err)
		}
	}

	return nil
}
