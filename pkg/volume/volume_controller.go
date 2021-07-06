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

package volume

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"k8s.io/klog"
)

type VolumeUpdateType int

const (
	directCSIVolumeKind = "DirectCSIVolume"

	VolumeUpdateTypeDeleting VolumeUpdateType = iota
	VolumeUpdateTypeUnknown
)

type DirectCSIVolumeListener struct {
	kubeClient      kubeclientset.Interface
	directcsiClient clientset.Interface
	nodeID          string
}

func (b *DirectCSIVolumeListener) InitializeKubeClient(k kubeclientset.Interface) {
	b.kubeClient = k
}

func (b *DirectCSIVolumeListener) InitializeDirectCSIClient(bc clientset.Interface) {
	b.directcsiClient = bc
}

func (b *DirectCSIVolumeListener) Add(ctx context.Context, obj *directcsi.DirectCSIVolume) error {
	return nil
}

func (b *DirectCSIVolumeListener) Update(ctx context.Context, old, new *directcsi.DirectCSIVolume) error {
	directCSIClient := b.directcsiClient.DirectV1beta2()
	dclient := directCSIClient.DirectCSIDrives()
	vclient := directCSIClient.DirectCSIVolumes()

	// Skip volumes from other nodes
	if new.Status.NodeName != b.nodeID {
		return nil
	}

	rmVolFromDrive := func(driveName string, volumeName string, capacity int64) error {
		drive, err := dclient.Get(ctx, driveName, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err != nil {
			return err
		}
		vFinalizer := directcsi.DirectCSIDriveFinalizerPrefix + volumeName

		dfinalizers := drive.GetFinalizers()

		// check if finalizer has already been removed
		found := false
		for _, df := range dfinalizers {
			if df == vFinalizer {
				found = true
				break
			}
		}
		if !found {
			return nil
		}

		// if not, remove finalizer
		updatedFinalizers := []string{}
		for _, df := range dfinalizers {
			if df == vFinalizer {
				continue
			}
			updatedFinalizers = append(updatedFinalizers, df)
		}
		if len(updatedFinalizers) == 1 {
			if updatedFinalizers[0] == directcsi.DirectCSIDriveFinalizerDataProtection {
				drive.Status.DriveStatus = directcsi.DriveStatusReady
			}
		}
		drive.SetFinalizers(updatedFinalizers)

		drive.Status.FreeCapacity = drive.Status.FreeCapacity + capacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity

		_, err = dclient.Update(ctx, drive, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err != nil {
			return err
		}
		return nil
	}

	cleanupVolume := func(vol *directcsi.DirectCSIVolume) error {
		if err := os.RemoveAll(vol.Status.HostPath); err != nil {
			return err
		}

		return rmVolFromDrive(vol.Status.Drive, vol.Name, vol.Status.TotalCapacity)
	}

	deleting := func() bool {
		if new.GetDeletionTimestamp().IsZero() {
			return false
		}
		return true
	}

	volumeUpdateType := func() VolumeUpdateType {
		if deleting() {
			return VolumeUpdateTypeDeleting
		}
		return VolumeUpdateTypeUnknown
	}
	switch volumeUpdateType() {
	case VolumeUpdateTypeDeleting:
		/*   ...from pkg/controller/controller.go
		 *
		 *   - if deletionTimestamp is set, i.e. the volume resource has
		 *     been deleted from the API
		 *       - cleanup the mount (not data deletion, just unmounting)
		 *
		 *       - the finalizer by volume name on the associated drive
		 *         is removed by the volume controller
		 *
		 *       - the purge protection finalizer is then removed by
		 *         the volume controller
		 */

		conditions := new.Status.Conditions
		for i, c := range conditions {
			switch c.Type {
			case string(directcsi.DirectCSIVolumeConditionReady):
			case string(directcsi.DirectCSIVolumeConditionStaged):
			case string(directcsi.DirectCSIVolumeConditionPublished):
				if conditions[i].Status == metav1.ConditionTrue {
					return fmt.Errorf("waiting for volume to be released before cleaning up")
				}
			}
		}

		if err := cleanupVolume(new); err != nil {
			return err
		}

		vfinalizers := new.GetFinalizers()
		updatedFinalizers := []string{}
		for _, vf := range vfinalizers {
			if vf == string(directcsi.DirectCSIVolumeFinalizerPurgeProtection) {
				continue
			}
			updatedFinalizers = append(updatedFinalizers, vf)
		}
		new.SetFinalizers(updatedFinalizers)

		_, err := vclient.Update(ctx, new, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		})
		if err != nil {
			return err
		}
	case VolumeUpdateTypeUnknown:
	}

	return nil
}

func (b *DirectCSIVolumeListener) Delete(ctx context.Context, obj *directcsi.DirectCSIVolume) error {
	return nil
}

func StartVolumeController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ctrl, err := listener.NewDefaultDirectCSIController("volume-controller", hostname, 40)
	if err != nil {
		klog.Error(err)
		return err
	}
	ctrl.AddDirectCSIVolumeListener(&DirectCSIVolumeListener{nodeID: nodeID})
	return ctrl.Run(ctx)
}

func SyncVolumes(ctx context.Context, nodeID string) {

	getVolumeLabels := func(ctx context.Context, vol *directcsi.DirectCSIVolume) map[string]string {
		var reservedDrivePath, reservedDriveName string
		driveClient := utils.GetDirectCSIClient().DirectCSIDrives()
		existingDrive, err := driveClient.Get(ctx, vol.Status.Drive, metav1.GetOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err == nil {
			vFinalizer := directcsi.DirectCSIDriveFinalizerPrefix + vol.Name
			dfinalizers := existingDrive.GetFinalizers()
			for _, df := range dfinalizers {
				if df == vFinalizer {
					reservedDrivePath = existingDrive.Status.Path
					reservedDriveName = utils.SanitizeLabelV(existingDrive.Name)
					break
				}
			}
		}

		volumeLabels := vol.ObjectMeta.GetLabels()
		if volumeLabels == nil {
			volumeLabels = make(map[string]string)
		}

		volumeLabels[directcsi.Group+"/node"] = vol.Status.NodeName
		volumeLabels[directcsi.Group+"/drive-path"] = filepath.Base(reservedDrivePath)
		volumeLabels[directcsi.Group+"/drive"] = reservedDriveName
		volumeLabels[directcsi.Group+"/created-by"] = "directcsi-controller"

		return volumeLabels
	}

	volumeClient := utils.GetDirectCSIClient().DirectCSIVolumes()
	volumeList, err := volumeClient.List(ctx, metav1.ListOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(),
	})
	if err != nil {
		klog.V(3).Infof("Error while syncing CRD versions in directcsivolume: %v", err)
		return
	}
	volumes := volumeList.Items
	for _, volume := range volumes {
		// Skip volumes from other nodes
		if volume.Status.NodeName != nodeID {
			continue
		}
		updateFunc := func() error {
			vol, err := volumeClient.Get(ctx, volume.Name, metav1.GetOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			})
			if err == nil {
				updateOpts := metav1.UpdateOptions{
					TypeMeta: utils.DirectCSIVolumeTypeMeta(),
				}
				// update the labels
				volumeLabels := getVolumeLabels(ctx, vol)
				vol.ObjectMeta.SetLabels(volumeLabels)
				_, err = volumeClient.Update(ctx, vol, updateOpts)
			}
			return err
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			klog.V(3).Infof("Error while syncing CRD versions in directcsivolume: %v", err)
		}
	}
}
