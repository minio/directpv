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
	"path/filepath"
	"syscall"

	direct_csi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	kubeclientset "k8s.io/client-go/kubernetes"
)

type DirectCSIDriveListener struct {
	kubeClient      kubeclientset.Interface
	directcsiClient clientset.Interface
	nodeID          string
}

func (b *DirectCSIDriveListener) InitializeKubeClient(k kubeclientset.Interface) {
	b.kubeClient = k
}

func (b *DirectCSIDriveListener) InitializeDirectCSIClient(bc clientset.Interface) {
	b.directcsiClient = bc
}

func (b *DirectCSIDriveListener) Add(ctx context.Context, obj *direct_csi.DirectCSIDrive) error {
	glog.V(1).Infof("add called for DirectCSIDrive %s", obj.Name)
	return nil
}

func (b *DirectCSIDriveListener) Update(ctx context.Context, old, new *direct_csi.DirectCSIDrive) error {
	directCSIClient := b.directcsiClient.DirectV1alpha1()
	var uErr error

	new, uErr = directCSIClient.DirectCSIDrives().Get(ctx, new.ObjectMeta.Name, metav1.GetOptions{})
	if uErr != nil {
		return uErr
	}

	if b.nodeID != new.Status.NodeName {
		glog.V(5).Infof("Skipping drive %s", new.ObjectMeta.Name)
		return nil
	}

	if new.Spec.RequestedFormat.Filesystem == "" && new.Spec.RequestedFormat.Mountpoint == "" {
		return nil
	}

	if new.Status.DriveStatus == direct_csi.Online {
		glog.Errorf("Cannot format a drive in use %s", new.ObjectMeta.Name)
		return nil
	}

	fsType := new.Spec.RequestedFormat.Filesystem
	if new.Status.Filesystem != "" && new.Status.Filesystem != "xfs" && fsType != "xfs" {
		glog.Errorf("Only xfs disks can be added - %s", new.ObjectMeta.Name)
		return nil
	}

	if fsType != "" {

		if fsType != "xfs" {
			glog.Errorf("Only xfs formatting is supported - %s", new.ObjectMeta.Name)
			return nil
		}

		isForceOptionSet := new.Spec.RequestedFormat.Force
		isPurgeOptionSet := new.Spec.RequestedFormat.Purge

		finalizers := new.ObjectMeta.GetFinalizers()
		if len(finalizers) > 0 {
			glog.Errorf("Cannot format the drive as the finalizers are not yet satisfied: %v", finalizers)
			return nil
		}

		if new.Status.AllocatedCapacity > 0 && !isPurgeOptionSet {
			glog.Errorf("Cannot format a used drive - %s. Set 'purge: true' to override", new.ObjectMeta.Name)
			return nil
		}

		if new.Status.Mountpoint != "" {
			if !isForceOptionSet {
				glog.Errorf("Cannot format a mounted drive - %s. Set 'force: true' to override", new.ObjectMeta.Name)
				return nil
			}
			// Get absolute path
			abMountPath, fErr := filepath.Abs(new.Status.Mountpoint)
			if fErr != nil {
				return fErr
			}
			// Unmount all mount refs to avoid later mounts to overlap earlier mounts
			if err := UnmountAllMountRefs(abMountPath); err != nil {
				return err
			}
			// Update the truth immediately that the drive is been unmounted (OR) the drive does not have a mountpoint
			new.Status.Mountpoint = ""
			if new, uErr = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); uErr != nil {
				return uErr
			}
		}
		if new.Status.Filesystem != "" && !isForceOptionSet {
			glog.Errorf("Drive already has a filesystem - %s", new.ObjectMeta.Name)
			return nil
		}
		if fErr := FormatDevice(ctx, new.Status.Path, fsType, isForceOptionSet); fErr != nil {
			return fmt.Errorf("Failed to format the device: %v", fErr)
		}

		// Update the truth immediately that the drive is been unmounted (OR) the drive does not have a mountpoint
		new.Status.Filesystem = fsType
		new.Status.DriveStatus = direct_csi.New
		new.Spec.RequestedFormat.Filesystem = ""
		new.Status.Mountpoint = ""
		new.Status.MountOptions = []string{}
		new.Status.AllocatedCapacity = 0
		if new, uErr = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); uErr != nil {
			return uErr
		}
	}

	if new.Status.Mountpoint == "" {
		mountPoint := new.Spec.RequestedFormat.Mountpoint
		if mountPoint == "" {
			mountPoint = filepath.Join(string(filepath.Separator), "mnt", "direct-csi", new.ObjectMeta.Name)
		}

		mountOptions := new.Spec.RequestedFormat.Mountoptions
		mountOptions = append(mountOptions, "prjquota")
		if err := MountDevice(new.Status.Path, mountPoint, fsType, mountOptions); err != nil {
			return fmt.Errorf("Failed to mount the device: %v", err)
		}

		new.Spec.RequestedFormat.Force = false
		new.Status.Mountpoint = mountPoint
		new.Spec.RequestedFormat.Mountpoint = ""
		new.Spec.RequestedFormat.Mountoptions = []string{}
		stat := &syscall.Statfs_t{}
		if err := syscall.Statfs(new.Status.Mountpoint, stat); err != nil {
			return err
		}
		availBlocks := int64(stat.Bavail)
		new.Status.FreeCapacity = int64(stat.Bsize) * availBlocks
		new.Status.AllocatedCapacity = 0

		if new, uErr = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); uErr != nil {
			return uErr
		}
		glog.V(4).Infof("Successfully mounted DirectCSIDrive %s", new.ObjectMeta.Name)
	}

	return nil
}

func (b *DirectCSIDriveListener) Delete(ctx context.Context, obj *direct_csi.DirectCSIDrive) error {
	return nil
}

func startDriveController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ctrl, err := listener.NewDefaultDirectCSIController("drive-controller", hostname, 40)
	if err != nil {
		glog.Error(err)
		return err
	}
	ctrl.AddDirectCSIDriveListener(&DirectCSIDriveListener{nodeID: nodeID})
	return ctrl.Run(ctx)
}
