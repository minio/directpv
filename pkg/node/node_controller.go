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
	"path/filepath"

	"github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/golang/glog"
	kubeclientset "k8s.io/client-go/kubernetes"
)

type DirectCSIDriveListener struct {
	kubeClient      kubeclientset.Interface
	directcsiClient clientset.Interface
}

func (b *DirectCSIDriveListener) InitializeKubeClient(k kubeclientset.Interface) {
	b.kubeClient = k
}

func (b *DirectCSIDriveListener) InitializeDirectCSIClient(bc clientset.Interface) {
	b.directcsiClient = bc
}

func (b *DirectCSIDriveListener) Add(ctx context.Context, obj *v1alpha1.DirectCSIDrive) error {
	glog.V(1).Infof("add called for DirectCSIDrive %s", obj.Name)
	return nil
}

func (b *DirectCSIDriveListener) Update(ctx context.Context, old, new *v1alpha1.DirectCSIDrive) error {
	glog.V(1).Infof("Update called for DirectCSIDrive %s", new.ObjectMeta.Name)

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		var gErr error
		directCSIClient := utils.GetDirectCSIClient()

		// Fetch the latest version in the queue
		new, gErr = directCSIClient.DirectCSIDrives().Get(ctx, new.ObjectMeta.Name, metav1.GetOptions{})
		if gErr != nil {
			return status.Error(codes.NotFound, gErr.Error())
		}

		if new.DriveStatus == v1alpha1.Unformatted {
			return status.Error(codes.Unimplemented, "Formatting drive is not yet implemented")
		}

		isReqSatisfiedAlready := func(old, new *v1alpha1.DirectCSIDrive) bool {
			return new.DriveStatus == v1alpha1.Online && (new.RequestedFormat.Mountpoint == "" || new.RequestedFormat.Mountpoint == old.Mountpoint)
		}

		// Do not process the request if satisfied already
		if (new.RequestedFormat == v1alpha1.RequestedFormat{} || !new.DirectCSIOwned || isReqSatisfiedAlready(old, new)) {
			return nil
		}

		var mountPoint string
		if new.RequestedFormat.Mountpoint == "" {
			mountPoint = filepath.Join("/mnt", new.ObjectMeta.Name)
		}

		// Mount the device to the mountpoint [Idempotent]
		mountOptions := []string{""}
		if err := MountDevice(new.Path, mountPoint, "", mountOptions); err != nil {
			return status.Errorf(codes.Internal, "Failed to format and mount the device: %v", err)
		}

		copiedDrive := new.DeepCopy()
		copiedDrive.Mountpoint = mountPoint
		copiedDrive.DriveStatus = v1alpha1.Online

		_, uErr := directCSIClient.DirectCSIDrives().Update(ctx, copiedDrive, metav1.UpdateOptions{})
		return uErr
	}); err != nil {
		return err
	}

	return nil
}

func (b *DirectCSIDriveListener) Delete(ctx context.Context, obj *v1alpha1.DirectCSIDrive) error {
	return nil
}

func startController(ctx context.Context) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ctrl, err := listener.NewDefaultDirectCSIController("node-controller", hostname, 40)
	if err != nil {
		glog.Error(err)
		return err
	}
	ctrl.AddDirectCSIDriveListener(&DirectCSIDriveListener{})
	return ctrl.Run(ctx)
}
