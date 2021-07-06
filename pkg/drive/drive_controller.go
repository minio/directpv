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

package drive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"

	"github.com/google/uuid"
	"k8s.io/klog"
)

type DriveUpdateType int

const (
	DriveUpdateTypeDelete DriveUpdateType = iota
	DriveUpdateTypeOwnAndFormat
	DriveUpdateTypeStorageSpace
	DriveUpdateTypeDriveParams
	DriveUpdateTypeVolumeDelete
	DriveUpdateTypeUnknown
)

type DirectCSIDriveListener struct {
	kubeClient      kubeclientset.Interface
	directcsiClient clientset.Interface
	nodeID          string
	mounter         sys.DriveMounter
	formatter       sys.DriveFormatter
	statter         sys.DriveStatter
}

func (b *DirectCSIDriveListener) InitializeKubeClient(k kubeclientset.Interface) {
	b.kubeClient = k
}

func (b *DirectCSIDriveListener) InitializeDirectCSIClient(bc clientset.Interface) {
	b.directcsiClient = bc
}

func (b *DirectCSIDriveListener) Add(ctx context.Context, obj *directcsi.DirectCSIDrive) error {
	return nil
}

func (d *DirectCSIDriveListener) Update(ctx context.Context, old, new *directcsi.DirectCSIDrive) error {
	var err error
	directCSIClient := d.directcsiClient.DirectV1beta2()

	// TODO: configure client to filter based on nodename
	if d.nodeID != new.Status.NodeName {
		return nil
	}
	klog.V(3).Infof("drive update called on %s", new.Name)

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

	ownAndFormat := func(ctx context.Context, old, new *directcsi.DirectCSIDrive) bool {
		// if directCSIOwned is set to true
		if new.Spec.DirectCSIOwned == true && old.Spec.DirectCSIOwned == false {
			return true
		}

		// if requested format is cleared
		if new.Spec.RequestedFormat == nil {
			return false
		}
		return true
	}

	storageSpace := func(ctx context.Context, old, new *directcsi.DirectCSIDrive) bool {
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

	driveParams := func(ctx context.Context, old, new *directcsi.DirectCSIDrive) bool {
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

	driveUpdateType := func(ctx context.Context, old, new *directcsi.DirectCSIDrive) DriveUpdateType {
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

	isDuplicateUUID := func(ctx context.Context, driveName, fsUUID string) (bool, error) {
		driveList, err := directCSIClient.DirectCSIDrives().List(ctx, metav1.ListOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		})
		if err != nil {
			return false, err
		}
		drives := driveList.Items

		for _, drive := range drives {
			if drive.Status.NodeName != d.nodeID || drive.Name == driveName {
				continue
			}
			if drive.Status.FilesystemUUID == fsUUID {
				return true, nil
			}
		}

		return false, nil
	}

	//TODO: volume purge logic
	var updateErr error
	switch driveUpdateType(ctx, old, new) {
	case DriveUpdateTypeDelete:
		if new.Status.DriveStatus != directcsi.DriveStatusTerminating {
			new.Status.DriveStatus = directcsi.DriveStatusTerminating
			if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
				return err
			}
		}

		finalizers := new.GetFinalizers()
		if len(finalizers) == 0 {
			return nil
		}

		if len(finalizers) > 1 {
			return fmt.Errorf("cannot delete drive in use")
		}
		finalizer := finalizers[0]

		if finalizer != directcsi.DirectCSIDriveFinalizerDataProtection {
			return fmt.Errorf("invalid state reached. Please contact subnet.min.io")
		}

		if err := sys.SafeUnmount(filepath.Join(sys.MountRoot, new.Name), nil); err != nil {
			return err
		}

		new.Finalizers = []string{}
		if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		}); err != nil {
			return err
		}
	case DriveUpdateTypeOwnAndFormat:
		klog.V(3).Infof("owning and formatting drive %s", new.Name)
		force := func() bool {
			if new.Spec.RequestedFormat != nil {
				return new.Spec.RequestedFormat.Force
			}
			return false
		}()
		mounted := new.Status.Mountpoint != ""
		formatted := new.Status.Filesystem != ""

		switch new.Status.DriveStatus {
		case directcsi.DriveStatusReleased:
			klog.V(3).Infof("rejected request to format a released drive %s", new.Name)
			return nil
		case directcsi.DriveStatusInUse:
			klog.V(3).Infof("rejected request to format a drive currently in use %s", new.Name)
			return nil
		case directcsi.DriveStatusUnavailable:
			klog.V(3).Infof("rejected request to format an unavailable drive %s", new.Name)
			return nil
		case directcsi.DriveStatusReady:
			klog.V(3).Infof("rejected request to format a ready drive %s", new.Name)
			return nil
		case directcsi.DriveStatusTerminating:
			klog.V(3).Infof("rejected request to format a terminating drive %s", new.Name)
			return nil
		case directcsi.DriveStatusAvailable:
			UUID := new.Status.FilesystemUUID
			if UUID == "" {
				UUID = uuid.New().String()
			} else {
				// Check if the FilesystemUUID is already taken by other drives in the same node
				isDup, err := isDuplicateUUID(ctx, new.Name, UUID)
				if err != nil {
					return err
				}
				if isDup {
					UUID = uuid.New().String()
				}
			}
			new.Status.FilesystemUUID = UUID

			directCSIPath := sys.GetDirectCSIPath(new.Status.FilesystemUUID)
			directCSIMount := filepath.Join(sys.MountRoot, new.Status.FilesystemUUID)
			if err := d.formatter.MakeBlockFile(directCSIPath, new.Status.MajorNumber, new.Status.MinorNumber); err != nil {
				klog.Error(err)
				updateErr = err
			}

			source := directCSIPath
			target := directCSIMount
			mountOpts := new.Spec.RequestedFormat.MountOptions
			if updateErr == nil {
				if !formatted || force {
					if mounted {
						if err := d.mounter.UnmountDrive(source); err != nil {
							err = fmt.Errorf("failed to unmount drive: %s %v", new.Name, err)
							klog.Error(err)
							updateErr = err
						} else {
							new.Status.Mountpoint = ""
							mounted = false
						}
					}

					if updateErr == nil {
						if err := d.formatter.FormatDrive(ctx, new.Status.FilesystemUUID, source, force); err != nil {
							err = fmt.Errorf("failed to format drive: %s %v", new.Name, err)
							klog.Error(err)
							updateErr = err
						} else {
							new.Status.Filesystem = string(sys.FSTypeXFS)
							new.Status.AllocatedCapacity = int64(0)
							formatted = true
						}
					}
				}
			}

			if updateErr == nil && !mounted {
				if err := d.mounter.MountDrive(source, target, mountOpts); err != nil {
					err = fmt.Errorf("failed to mount drive: %s %v", new.Name, err)
					klog.Error(err)
					updateErr = err
				} else {
					new.Status.Mountpoint = target
					new.Status.MountOptions = mountOpts
					freeCapacity, sErr := d.statter.GetFreeCapacityFromStatfs(new.Status.Mountpoint)
					if sErr != nil {
						klog.Error(sErr)
						updateErr = sErr
					} else {
						mounted = true
						new.Status.FreeCapacity = freeCapacity
						new.Status.AllocatedCapacity = new.Status.TotalCapacity - new.Status.FreeCapacity
					}
				}
			}

			conditions := new.Status.Conditions
			for i, c := range conditions {
				switch c.Type {
				case string(directcsi.DirectCSIDriveConditionOwned):
					conditions[i].Status = utils.BoolToCondition(formatted && mounted)
					if formatted && mounted {
						conditions[i].Reason = string(directcsi.DirectCSIDriveReasonAdded)
						conditions[i].LastTransitionTime = metav1.Now()
					}
					conditions[i].Message = ""
					if updateErr != nil {
						conditions[i].Message = updateErr.Error()
					}
				case string(directcsi.DirectCSIDriveConditionMounted):
					conditions[i].Status = utils.BoolToCondition(mounted)
					conditions[i].Reason = string(directcsi.DirectCSIDriveReasonAdded)
					conditions[i].LastTransitionTime = metav1.Now()
					conditions[i].Message = func() string {
						if conditions[i].Status == metav1.ConditionTrue {
							return string(directcsi.DirectCSIDriveMessageMounted)
						}
						return string(directcsi.DirectCSIDriveMessageNotMounted)
					}()
				case string(directcsi.DirectCSIDriveConditionFormatted):
					conditions[i].Status = utils.BoolToCondition(formatted)
					conditions[i].Reason = string(directcsi.DirectCSIDriveReasonAdded)
					conditions[i].LastTransitionTime = metav1.Now()
					conditions[i].Message = func() string {
						if conditions[i].Status == metav1.ConditionTrue {
							return string(directcsi.DirectCSIDriveMessageFormatted)
						}
						return string(directcsi.DirectCSIDriveMessageNotFormatted)
					}()
				}
			}
			if updateErr == nil {
				new.Finalizers = []string{
					directcsi.DirectCSIDriveFinalizerDataProtection,
				}
				new.Status.DriveStatus = directcsi.DriveStatusReady
				new.Spec.RequestedFormat = nil
			}

			if new, err = directCSIClient.DirectCSIDrives().Update(ctx, new, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
			}); err != nil {
				return err
			}
			return nil
		}
	case DriveUpdateTypeStorageSpace:
		// no-op
	case DriveUpdateTypeDriveParams:
		// no-op
	default:
		return updateErr
	}
	return nil
}

func (b *DirectCSIDriveListener) Delete(ctx context.Context, obj *directcsi.DirectCSIDrive) error {
	return nil
}

func StartDriveController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	ctrl, err := listener.NewDefaultDirectCSIController("drive-controller", hostname, 40)
	if err != nil {
		klog.Error(err)
		return err
	}
	ctrl.AddDirectCSIDriveListener(&DirectCSIDriveListener{
		nodeID:    nodeID,
		mounter:   &sys.DefaultDriveMounter{},
		formatter: &sys.DefaultDriveFormatter{},
		statter:   &sys.DefaultDriveStatter{},
	})
	return ctrl.Run(ctx)
}
