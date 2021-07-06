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
	"reflect"
	"sort"

	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	kexec "k8s.io/utils/exec"
	"k8s.io/utils/mount"

	"k8s.io/klog"
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
		klog.V(5).Infof("Error while reading the disk format: (%s)", err.Error())
	}
	return fs, err
}

// AddDriveFinalizersWithConflictRetry - appends a finalizer to the csidrive's finalizers list
func AddDriveFinalizersWithConflictRetry(ctx context.Context, csiDriveName string, finalizers []string, crdVersion string) error {
	directCSIClient := utils.GetDirectCSIClient()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, csiDriveName, metav1.GetOptions{
			TypeMeta: utils.NewTypeMeta(crdVersion, "DirectCSIDrive"),
		})
		if dErr != nil {
			return dErr
		}
		copiedDrive := csiDrive.DeepCopy()
		for _, finalizer := range finalizers {
			copiedDrive.ObjectMeta.SetFinalizers(utils.AddFinalizer(&copiedDrive.ObjectMeta, finalizer))
		}
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{
			TypeMeta: utils.NewTypeMeta(crdVersion, "DirectCSIDrive"),
		})
		return err
	}); err != nil {
		klog.V(5).Infof("Error while adding finalizers to csidrive: (%s)", err.Error())
		return err
	}
	return nil
}

// RemoveDriveFinalizerWithConflictRetry - removes a finalizer from the csidrive's finalizers list
func RemoveDriveFinalizerWithConflictRetry(ctx context.Context, csiDriveName string, finalizer, crdVersion string) error {
	directCSIClient := utils.GetDirectCSIClient()
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		csiDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, csiDriveName, metav1.GetOptions{
			TypeMeta: utils.NewTypeMeta(crdVersion, "DirectCSIDrive"),
		})
		if dErr != nil {
			return dErr
		}
		copiedDrive := csiDrive.DeepCopy()
		copiedDrive.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&copiedDrive.ObjectMeta, finalizer))
		_, err := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{
			TypeMeta: utils.NewTypeMeta(crdVersion, "DirectCSIDrive"),
		})
		return err
	}); err != nil {
		klog.V(5).Infof("Error while adding finalizers to csidrive: (%s)", err.Error())
		return err
	}
	return nil
}

func CheckStatusEquality(existingConditions, newConditions []metav1.Condition) bool {
	extractStatuses := func(conds []metav1.Condition) []metav1.ConditionStatus {
		condStatuses := []metav1.ConditionStatus{}
		for _, cond := range conds {
			condStatuses = append(condStatuses, cond.Status)
		}
		return condStatuses
	}

	return reflect.DeepEqual(extractStatuses(existingConditions), extractStatuses(newConditions))
}
