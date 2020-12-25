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

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/dev"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
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

func (b *DirectCSIVolumeListener) Add(ctx context.Context, obj *v1alpha1.DirectCSIVolume) error {
	glog.V(1).Infof("add called for DirectCSIVolume %s", obj.Name)
	return nil
}

func (b *DirectCSIVolumeListener) Update(ctx context.Context, old, new *v1alpha1.DirectCSIVolume) error {
	glog.V(1).Infof("Update called for DirectCSIVolume %s", new.ObjectMeta.Name)

	directCSIClient := b.directcsiClient.DirectV1alpha1()
	if utils.CheckVolumeStatusCondition(new.Status.Conditions, "published", metav1.ConditionTrue) {
		sc := utils.GetVolumeStatusCondition(new.Status.Conditions, "volumestats")
		duration := time.Since(sc.LastTransitionTime.Time)
		if duration > 5*time.Minute {
			xfsQuota := &dev.XFSQuota{
				Path:      new.Status.ContainerPath,
				ProjectID: new.ObjectMeta.Name,
			}
			volStats, err := xfsQuota.GetVolumeStats(ctx)
			if err != nil {
				return err
			}
			new.Status.TotalCapacity = volStats.TotalBytes
			new.Status.AvailableCapacity = volStats.AvailableBytes
			new.Status.UsedCapacity = volStats.UsedBytes
			utils.UpdateVolumeStatusCondition(new.Status.Conditions, "volumestats", metav1.ConditionTrue)
			if _, vErr := directCSIClient.DirectCSIVolumes().Update(ctx, new, metav1.UpdateOptions{}); vErr != nil {
				return vErr
			}
		}
	}

	if !new.ObjectMeta.GetDeletionTimestamp().IsZero() {
		finalizers := new.ObjectMeta.GetFinalizers()
		if len(finalizers) > 0 {
			if len(finalizers) == 1 && finalizers[0] == fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, "purge-protection") {
				if new.Status.OwnerDrive != "" {
					if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						ownerDrive, dErr := directCSIClient.DirectCSIDrives().Get(ctx, new.Status.OwnerDrive, metav1.GetOptions{})
						if dErr != nil {
							return dErr
						}
						copiedDrive := ownerDrive.DeepCopy()
						copiedDrive.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&copiedDrive.ObjectMeta, fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, new.ObjectMeta.Name)))
						if len(copiedDrive.ObjectMeta.Finalizers) == 0 {
							copiedDrive.Status.DriveStatus = v1alpha1.Terminating // || ""
							copiedDrive.Spec.DirectCSIOwned = false               // Format and make it fresh
						}
						if _, dErr = directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{}); dErr != nil {
							return dErr
						}
						return nil
					}); err != nil {
						return err
					}
				}
				// Umount the container path
				if mErr := utils.UnmountIfMounted(new.Status.ContainerPath); mErr != nil {
					return mErr
				}
				// Umount the staging path
				if mErr := utils.UnmountIfMounted(new.Status.StagingPath); mErr != nil {
					return mErr
				}
				// Unset the owner drive
				new.Status.OwnerDrive = ""
				new.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&new.ObjectMeta, fmt.Sprintf("%s/%s", v1alpha1.SchemeGroupVersion.Group, "purge-protection")))
				if _, vErr := directCSIClient.DirectCSIVolumes().Update(ctx, new, metav1.UpdateOptions{}); vErr != nil {
					return vErr
				}
			}
		}
	}

	return nil
}

func (b *DirectCSIVolumeListener) Delete(ctx context.Context, obj *v1alpha1.DirectCSIVolume) error {
	glog.V(1).Infof("Delete called for DirectCSIVolume %s", obj.ObjectMeta.Name)
	return nil
}

func startVolumeController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ctrl, err := listener.NewDefaultDirectCSIController("volume-controller", hostname, 40)
	if err != nil {
		glog.Error(err)
		return err
	}
	ctrl.AddDirectCSIVolumeListener(&DirectCSIVolumeListener{nodeID: nodeID})
	return ctrl.Run(ctx)
}
