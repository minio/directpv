// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/klog/v2"
)

type volumeEventHandler struct {
	nodeID      string
	safeUnmount func(target string, force, detach, expire bool) error
}

// StartController starts volume controller.
func StartController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(newVolumeEventHandler(nodeID), "volume-controller", hostname, 40)
	return listener.Run(ctx)
}

func newVolumeEventHandler(nodeID string) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID:      nodeID,
		safeUnmount: mount.SafeUnmount,
	}
}

func (handler *volumeEventHandler) ListerWatcher() cache.ListerWatcher {
	return client.VolumesListerWatcher(handler.nodeID)
}

func (handler *volumeEventHandler) KubeClient() kubernetes.Interface {
	return client.GetKubeClient()
}

func (handler *volumeEventHandler) Name() string {
	return "volume"
}

func (handler *volumeEventHandler) ObjectType() runtime.Object {
	return &directcsi.DirectCSIVolume{}
}

func (handler *volumeEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	volume, err := client.GetLatestDirectCSIVolumeInterface().Get(
		ctx, args.Object.(*directcsi.DirectCSIVolume).Name, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
	)
	if err != nil {
		return err
	}
	if !volume.GetDeletionTimestamp().IsZero() {
		return handler.delete(ctx, volume)
	}
	return nil
}

func (handler *volumeEventHandler) delete(ctx context.Context, volume *directcsi.DirectCSIVolume) error {
	finalizers, _ := excludeFinalizer(
		volume.GetFinalizers(), string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
	)
	if len(finalizers) > 0 {
		return fmt.Errorf("waiting for the volume to be released before cleaning up")
	}

	if volume.Status.ContainerPath != "" {
		if err := handler.safeUnmount(volume.Status.ContainerPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.InfoS("failed to umount container path",
					"volume", volume.Name,
					"containerPath", volume.Status.ContainerPath,
					"error", err,
				)
				return err
			}
		}
	}
	if volume.Status.StagingPath != "" {
		if err := handler.safeUnmount(volume.Status.StagingPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.InfoS("failed to umount staging path",
					"volume", volume.Name,
					"containerPath", volume.Status.StagingPath,
					"error", err,
				)
				return err
			}
		}
	}

	// Remove associated directory of the volume.
	if err := os.RemoveAll(volume.Status.HostPath); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			klog.InfoS("failed to remove host path",
				"volume", volume.Name,
				"hostPath", volume.Status.HostPath,
				"error", err,
			)
			return err
		}
	}

	// Release volume from associated drive.
	if err := handler.releaseVolume(ctx, volume.Status.Drive, volume.Name, volume.Status.TotalCapacity); err != nil {
		return err
	}

	volume.SetFinalizers(finalizers)
	_, err := client.GetLatestDirectCSIVolumeInterface().Update(
		ctx, volume, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
	)

	return err
}

func (handler *volumeEventHandler) releaseVolume(ctx context.Context, driveName, volumeName string, capacity int64) error {
	driveInterface := client.GetLatestDirectCSIDriveInterface()
	drive, err := driveInterface.Get(
		ctx, driveName, metav1.GetOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
	)
	if err != nil {
		return err
	}

	finalizers, found := excludeFinalizer(
		drive.GetFinalizers(), directcsi.DirectCSIDriveFinalizerPrefix+volumeName,
	)

	if found {
		if len(finalizers) == 1 {
			if finalizers[0] == directcsi.DirectCSIDriveFinalizerDataProtection {
				drive.Status.DriveStatus = directcsi.DriveStatusReady
			}
		}

		drive.SetFinalizers(finalizers)
		drive.Status.FreeCapacity += capacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity

		_, err = driveInterface.Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
		)
	}

	return err
}

func excludeFinalizer(finalizers []string, finalizer string) (result []string, found bool) {
	for _, f := range finalizers {
		if f != finalizer {
			result = append(result, f)
		} else {
			found = true
		}
	}
	return
}
