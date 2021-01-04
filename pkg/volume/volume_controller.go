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

package volume

import (
	"context"
	"fmt"
	"os"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

type VolumeUpdateType int

const (
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

func (b *DirectCSIVolumeListener) Add(ctx context.Context, obj *directv1alpha1.DirectCSIVolume) error {
	return nil
}

func (b *DirectCSIVolumeListener) Update(ctx context.Context, old, new *directv1alpha1.DirectCSIVolume) error {
	directCSIClient := utils.GetDirectCSIClient()
	dclient := directCSIClient.DirectCSIDrives()
	vclient := directCSIClient.DirectCSIVolumes()

	// Skip volumes from other nodes
	if new.Status.NodeName != b.nodeID {
		return nil
	}
	rmVolFinalizerFromDrive := func(driveName string, volumeName string) error {
		drive, err := dclient.Get(ctx, driveName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		vFinalizer := directv1alpha1.DirectCSIDriveFinalizerPrefix + volumeName

		dfinalizers := drive.GetFinalizers()
		updatedFinalizers := []string{}
		for _, df := range dfinalizers {
			if df == vFinalizer {
				continue
			}
			updatedFinalizers = append(updatedFinalizers, df)
		}
		drive.SetFinalizers(updatedFinalizers)

		_, err = dclient.Update(ctx, drive, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
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
			case string(directv1alpha1.DirectCSIVolumeConditionStaged):
				fallthrough
			case string(directv1alpha1.DirectCSIVolumeConditionPublished):
				if conditions[i].Status == metav1.ConditionTrue {
					return fmt.Errorf("waiting for volume to be released before cleaning up")
				}
			}
		}

		if err := rmVolFinalizerFromDrive(new.Status.Drive, new.Name); err != nil {
			return err
		}

		vfinalizers := new.GetFinalizers()
		updatedFinalizers := []string{}
		for _, vf := range vfinalizers {
			if vf == string(directv1alpha1.DirectCSIVolumeFinalizerPurgeProtection) {
				continue
			}
			updatedFinalizers = append(updatedFinalizers, vf)
		}
		new.SetFinalizers(updatedFinalizers)

		_, err := vclient.Update(ctx, new, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	case VolumeUpdateTypeUnknown:
	}

	return nil
}

func (b *DirectCSIVolumeListener) Delete(ctx context.Context, obj *directv1alpha1.DirectCSIVolume) error {
	return nil
}

func StartVolumeController(ctx context.Context, nodeID string) error {
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
