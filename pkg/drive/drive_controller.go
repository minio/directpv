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

package drive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	directv1alpha1 "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/sys"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

type DriveUpdateType int

const (
	DriveUpdateTypeDelete DriveUpdateType = iota
	DriveUpdateTypeOwnAndFormat
	DriveUpdateTypeStorageSpace
	DriveUpdateTypeDriveParams
	DriveUpdateTypeUnknown
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

func (b *DirectCSIDriveListener) Add(ctx context.Context, obj *directv1alpha1.DirectCSIDrive) error {
	return nil
}

func (d *DirectCSIDriveListener) Update(ctx context.Context, old, new *directv1alpha1.DirectCSIDrive) error {
	var err error
	directCSIClient := d.directcsiClient.DirectV1alpha1()

	// TODO: configure client to filter based on nodename
	if d.nodeID != new.Status.NodeName {
		return nil
	}
	glog.V(3).Infof("drive update called on %s", new.Name)

	// Determine the type of update
	// - Own drive & Format
	// - Update free and Allocated space values
	// - Changes to other parameters such as drive path

	deleting := func() bool {
		if new.GetDeletionTimestamp().IsZero() {
			return false
		}
		return true
	}

	ownAndFormat := func(ctx context.Context, old, new *directv1alpha1.DirectCSIDrive) bool {
		// if directCSIOwned is set to true
		if new.Spec.DirectCSIOwned == true && old.Spec.DirectCSIOwned == false {
			return true
		}

		// if requested format is cleared
		if new.Spec.RequestedFormat == nil {
			return false
		}

		// if requested format changes
		if new.Spec.RequestedFormat != nil && old.Spec.RequestedFormat == nil {
			return true
		}
		// if filesystem is changed
		if new.Spec.RequestedFormat.Filesystem != old.Spec.RequestedFormat.Filesystem {
			return true
		}
		// if force is set
		if new.Spec.RequestedFormat.Force && !old.Spec.RequestedFormat.Force {
			return true
		}
		return false
	}

	storageSpace := func(ctx context.Context, old, new *directv1alpha1.DirectCSIDrive) bool {
		// if total, allocated or free capacity changes
		if new.Status.TotalCapacity != old.Status.TotalCapacity {
			return true
		}
		if new.Status.AllocatedCapacity != old.Status.AllocatedCapacity {
			return true
		}
		if new.Status.FreeCapacity != old.Status.FreeCapacity {
			return true
		}
		return false
	}

	driveParams := func(ctx context.Context, old, new *directv1alpha1.DirectCSIDrive) bool {
		// if drivePath or partition number changes after reboot
		if new.Status.Path != old.Status.Path {
			return true
		}
		if new.Status.PartitionNum != old.Status.PartitionNum {
			return true
		}
		if new.Status.RootPartition != old.Status.RootPartition {
			return true
		}
		return false
	}

	driveUpdateType := func(ctx context.Context, old, new *directv1alpha1.DirectCSIDrive) DriveUpdateType {
		if deleting() {
			return DriveUpdateTypeDelete
		}
		if ownAndFormat(ctx, old, new) {
			return DriveUpdateTypeOwnAndFormat
		}
		if storageSpace(ctx, old, new) {
			return DriveUpdateTypeStorageSpace
		}
		if driveParams(ctx, old, new) {
			return DriveUpdateTypeDriveParams
		}
		return DriveUpdateTypeUnknown
	}

	//TODO: volume purge logic
	var updateErr error
	switch driveUpdateType(ctx, old, new) {
	case DriveUpdateTypeDelete:
		if new.Status.DriveStatus != directv1alpha1.DriveStatusTerminating {
			new.Status.DriveStatus = directv1alpha1.DriveStatusTerminating
			if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}

		finalizers := new.GetFinalizers()
		if len(finalizers) > 1 {
			return fmt.Errorf("cannot delete drive in use")
		}
		finalizer := finalizers[0]

		if finalizer != directv1alpha1.DirectCSIDriveFinalizerDataProtection {
			return fmt.Errorf("invalid state reached. Please contact subnet.min.io")
		}

		if err := sys.SafeUnmount(filepath.Join(sys.MountRoot, new.Name), nil); err != nil {
			return err
		}

		new.Finalizers = []string{}
		if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); err != nil {
			return err
		}
	case DriveUpdateTypeOwnAndFormat:
		glog.V(3).Infof("owning and formatting drive %s", new.Name)
		force := func() bool {
			if new.Spec.RequestedFormat != nil {
				return new.Spec.RequestedFormat.Force
			}
			return false
		}()
		mounted := new.Status.Mountpoint != ""
		formatted := new.Status.Filesystem == ""

		source := new.Status.Path
		target := filepath.Join(sys.MountRoot, new.Name)
		mountOpts := new.Spec.RequestedFormat.MountOptions

		switch new.Status.DriveStatus {
		case directv1alpha1.DriveStatusInUse:
			glog.V(2).Infof("rejected request to format a drive currently in use %s", new.Name)
			return nil
		case directv1alpha1.DriveStatusUnavailable:
			glog.V(2).Infof("rejected request to format an unavailable drive %s", new.Name)
			return nil
		case directv1alpha1.DriveStatusReady:
			glog.V(2).Infof("rejected request to format a ready drive %s", new.Name)
			return nil
		case directv1alpha1.DriveStatusTerminating:
			glog.V(2).Infof("rejected request to format a terminating drive %s", new.Name)
			return nil
		case directv1alpha1.DriveStatusAvailable:
			if !formatted || force {
				if mounted {
					if err := unmountDrive(target); err != nil {
						glog.Errorf("failed to unmount drive: %s %v", new.Name, err)
						return err
					}
				}
				mounted = false
				if err := formatDrive(ctx, source); err != nil {
					glog.Errorf("failed to format drive: %s %v", new.Name, err)
					return err
				}
				formatted = true
			}

			if !mounted {
				if err := mountDrive(source, target, mountOpts); err != nil {
					glog.Errorf("failed to mount drive: %s %v", new.Name, err)
					return err
				}
				mounted = true
			}

			conditions := new.Status.Conditions
			for i, c := range conditions {
				switch c.Type {
				case string(directv1alpha1.DirectCSIDriveConditionOwned):
					conditions[i].Status = utils.BoolToCondition(formatted && mounted)
					if formatted && mounted {
						conditions[i].Reason = string(directv1alpha1.DirectCSIDriveReasonAdded)
						conditions[i].LastTransitionTime = metav1.Now()
					}
				case string(directv1alpha1.DirectCSIDriveConditionMounted):
					conditions[i].Status = utils.BoolToCondition(mounted)
					conditions[i].Reason = string(directv1alpha1.DirectCSIDriveReasonAdded)
					conditions[i].LastTransitionTime = metav1.Now()
				case string(directv1alpha1.DirectCSIDriveConditionFormatted):
					conditions[i].Status = utils.BoolToCondition(formatted)
					conditions[i].Reason = string(directv1alpha1.DirectCSIDriveReasonAdded)
					conditions[i].LastTransitionTime = metav1.Now()
				}
			}
			new.Finalizers = []string{
				directv1alpha1.DirectCSIDriveFinalizerDataProtection,
			}
			new.Status.DriveStatus = directv1alpha1.DriveStatusReady
			new.Status.Mountpoint = target
			new.Status.MountOptions = mountOpts
			new.Spec.RequestedFormat = nil

			if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{}); err != nil {
				return err
			}
			return nil
		}
	case DriveUpdateTypeStorageSpace:
		glog.V(3).Infof("drive update storage space: //no-op")
		// no-op
	case DriveUpdateTypeDriveParams:
		glog.V(3).Infof("drive update drive params: //no-op")
		// no-op
	default:
		glog.V(3).Infof("unknown update type: %s", new.Name)
		return updateErr
	}
	return nil
}

func (b *DirectCSIDriveListener) Delete(ctx context.Context, obj *directv1alpha1.DirectCSIDrive) error {
	return nil
}

func StartDriveController(ctx context.Context, nodeID string) error {
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
