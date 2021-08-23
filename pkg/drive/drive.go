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

	"github.com/google/uuid"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/listener"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type DriveEventHandler struct {
	directCSIClient clientset.Interface
	kubeClient      kubernetes.Interface
	nodeID          string
	mounter         sys.DriveMounter
	formatter       sys.DriveFormatter
	statter         sys.DriveStatter
}

func NewDriveEventHandler(nodeID string) *DriveEventHandler {
	return &DriveEventHandler{
		directCSIClient: utils.GetDirectClientset(),
		kubeClient:      utils.GetKubeClient(),
		nodeID:          nodeID,
		mounter:         &sys.DefaultDriveMounter{},
		formatter:       &sys.DefaultDriveFormatter{},
		statter:         &sys.DefaultDriveStatter{},
	}
}

func (handler *DriveEventHandler) ListerWatcher() cache.ListerWatcher {
	labelSelector := ""
	if handler.nodeID != "" {
		labelSelector = fmt.Sprintf("%s=%s", utils.NodeLabel, utils.SanitizeLabelV(handler.nodeID))
	}

	optionsModifier := func(options *metav1.ListOptions) {
		options.LabelSelector = labelSelector
	}

	return cache.NewFilteredListWatchFromClient(
		handler.directCSIClient.DirectV1beta2().RESTClient(),
		"DirectCSIDrives",
		"",
		optionsModifier,
	)
}

func (handler *DriveEventHandler) KubeClient() kubernetes.Interface {
	return handler.kubeClient
}

func (handler *DriveEventHandler) Name() string {
	return "drive"
}

func (handler *DriveEventHandler) ObjectType() runtime.Object {
	return &directcsi.DirectCSIDrive{}
}

func (handler *DriveEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.AddEvent, listener.UpdateEvent:
		return handler.update(ctx, args.Object.(*directcsi.DirectCSIDrive))
	case listener.DeleteEvent:
		return handler.delete(ctx, args.Object.(*directcsi.DirectCSIDrive))
	}
	return nil
}

func (handler *DriveEventHandler) getFSUUID(ctx context.Context, drive *directcsi.DirectCSIDrive) (string, error) {
	if drive.Status.FilesystemUUID == "" {
		return uuid.New().String(), nil
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	// Use new UUID if it is aleady used in another drive.
	resultCh, err := utils.ListDrives(
		ctx, handler.directCSIClient.DirectV1beta2().DirectCSIDrives(), []string{handler.nodeID}, nil, nil, utils.MaxThreadCount,
	)
	if err != nil {
		return "", err
	}

	for result := range resultCh {
		if result.Err != nil {
			return "", err
		}

		if result.Drive.Name != drive.Name && result.Drive.Status.FilesystemUUID == drive.Status.FilesystemUUID {
			return uuid.New().String(), nil
		}
	}

	return drive.Status.FilesystemUUID, nil
}

func (handler *DriveEventHandler) format(ctx context.Context, drive *directcsi.DirectCSIDrive) (err error) {
	fsUUID, err := handler.getFSUUID(ctx, drive)
	if err != nil {
		klog.Error(err)
		return err
	}
	drive.Status.FilesystemUUID = fsUUID

	source := sys.GetDirectCSIPath(drive.Status.FilesystemUUID)
	if err = handler.formatter.MakeBlockFile(source, drive.Status.MajorNumber, drive.Status.MinorNumber); err != nil {
		klog.Error(err)
	}

	target := filepath.Join(sys.MountRoot, drive.Status.FilesystemUUID)
	mountOpts := drive.Spec.RequestedFormat.MountOptions
	force := drive.Spec.RequestedFormat.Force
	mounted := drive.Status.Mountpoint != ""
	formatted := drive.Status.Filesystem != ""

	if err == nil && (!formatted || force) {
		if mounted {
			if err = handler.mounter.UnmountDrive(source); err != nil {
				err = fmt.Errorf("failed to unmount drive %s; %w", drive.Name, err)
				klog.Error(err)
			} else {
				drive.Status.Mountpoint = ""
				mounted = false
			}
		}

		if err == nil {
			if err = handler.formatter.FormatDrive(ctx, drive.Status.FilesystemUUID, source, force); err != nil {
				err = fmt.Errorf("failed to format drive %s; %w", drive.Name, err)
				klog.Error(err)
			} else {
				drive.Status.Filesystem = string(sys.FSTypeXFS)
				drive.Status.AllocatedCapacity = 0
				formatted = true
			}
		}
	}

	if err == nil && formatted && !mounted {
		if err = handler.mounter.MountDrive(source, target, mountOpts); err != nil {
			err = fmt.Errorf("failed to mount drive %s; %w", drive.Name, err)
			klog.Error(err)
		} else {
			drive.Status.Mountpoint = target
			drive.Status.MountOptions = mountOpts
			freeCapacity := int64(0)
			if freeCapacity, err = handler.statter.GetFreeCapacityFromStatfs(drive.Status.Mountpoint); err != nil {
				klog.Error(err)
			} else {
				mounted = true
				drive.Status.FreeCapacity = freeCapacity
				drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity
			}
		}
	}

	message := ""
	if err != nil {
		message = err.Error()
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionOwned),
		utils.BoolToCondition(formatted && mounted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	message = string(directcsi.DirectCSIDriveMessageNotMounted)
	if mounted {
		message = string(directcsi.DirectCSIDriveMessageMounted)
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionMounted),
		utils.BoolToCondition(mounted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	message = string(directcsi.DirectCSIDriveMessageNotFormatted)
	if formatted {
		message = string(directcsi.DirectCSIDriveMessageFormatted)
	}
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionFormatted),
		utils.BoolToCondition(formatted),
		string(directcsi.DirectCSIDriveReasonAdded),
		message,
	)

	if err == nil {
		drive.Finalizers = []string{directcsi.DirectCSIDriveFinalizerDataProtection}
		drive.Status.DriveStatus = directcsi.DriveStatusReady
		drive.Spec.RequestedFormat = nil
	}

	_, err = handler.directCSIClient.DirectV1beta2().DirectCSIDrives().Update(
		ctx, drive, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		},
	)

	return err
}

func (handler *DriveEventHandler) update(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	klog.V(5).Infof("drive update called on %s", drive.Name)

	if drive.Spec.DirectCSIOwned && drive.Spec.RequestedFormat != nil {
		klog.V(3).Infof("owning and formatting drive %s", drive.Name)

		switch drive.Status.DriveStatus {
		case directcsi.DriveStatusAvailable:
			return handler.format(ctx, drive)
		case directcsi.DriveStatusReleased,
			directcsi.DriveStatusUnavailable,
			directcsi.DriveStatusReady,
			directcsi.DriveStatusTerminating,
			directcsi.DriveStatusInUse:
			klog.V(3).Infof("rejecting to format drive %s due to %s", drive.Name, drive.Status.DriveStatus)
		}
	}

	return nil
}

func (handler *DriveEventHandler) delete(ctx context.Context, drive *directcsi.DirectCSIDrive) error {
	directCSIClient := handler.directCSIClient.DirectV1beta2()
	if drive.Status.DriveStatus != directcsi.DriveStatusTerminating {
		drive.Status.DriveStatus = directcsi.DriveStatusTerminating
		if _, err := directCSIClient.DirectCSIDrives().Update(ctx, drive, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		}); err != nil {
			return err
		}
	}

	finalizers := drive.GetFinalizers()
	if len(finalizers) == 0 {
		return nil
	}

	if len(finalizers) > 1 {
		return fmt.Errorf("cannot delete drive in use")
	}
	finalizer := finalizers[0]

	if finalizer != directcsi.DirectCSIDriveFinalizerDataProtection {
		return fmt.Errorf("invalid state reached. Report this issue at https://github.com/minio/direct-csi/issues")
	}

	if err := sys.SafeUnmount(filepath.Join(sys.MountRoot, drive.Name), nil); err != nil {
		return err
	}

	drive.Finalizers = []string{}
	_, err := directCSIClient.DirectCSIDrives().Update(
		ctx, drive, metav1.UpdateOptions{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
		},
	)
	return err
}

func StartController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(NewDriveEventHandler(nodeID), "drive-controller", hostname, 40)
	return listener.Run(ctx)
}
