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

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

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

type volumeEventHandler struct {
	nodeID      string
	safeUnmount func(target string, force, detach, expire bool) error
}

func newVolumeEventHandler(nodeID string) *volumeEventHandler {
	return &volumeEventHandler{
		nodeID:      nodeID,
		safeUnmount: sys.Unmount,
	}
}

func (handler *volumeEventHandler) ListerWatcher() cache.ListerWatcher {
	return VolumesListerWatcher(handler.nodeID)
}

func (handler *volumeEventHandler) Name() string {
	return "volume"
}

func (handler *volumeEventHandler) ObjectType() runtime.Object {
	return &types.Volume{}
}

func (handler *volumeEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	volume, err := client.VolumeClient().Get(
		ctx, args.Object.(*types.Volume).Name, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)
	if err != nil {
		return err
	}
	if !volume.GetDeletionTimestamp().IsZero() {
		return handler.delete(ctx, volume)
	}
	return nil
}

func (handler *volumeEventHandler) delete(ctx context.Context, volume *types.Volume) error {
	finalizers, _ := excludeFinalizer(
		volume.GetFinalizers(), string(consts.VolumeFinalizerPurgeProtection),
	)
	if len(finalizers) > 0 {
		return fmt.Errorf("waiting for the volume to be released before cleaning up")
	}

	if volume.Status.ContainerPath != "" {
		if err := handler.safeUnmount(volume.Status.ContainerPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.ErrorS(err, "unable to unmount container path",
					"volume", volume.Name,
					"containerPath", volume.Status.ContainerPath,
				)
				return err
			}
		}
	}
	if volume.Status.StagingPath != "" {
		if err := handler.safeUnmount(volume.Status.StagingPath, true, true, false); err != nil {
			if _, ok := err.(*os.PathError); !ok {
				klog.ErrorS(err, "unable to unmount staging path",
					"volume", volume.Name,
					"containerPath", volume.Status.StagingPath,
				)
				return err
			}
		}
	}

	// Remove associated directory of the volume.
	if err := os.RemoveAll(volume.Status.HostPath); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			klog.ErrorS(err, "unable to remove host path",
				"volume", volume.Name,
				"hostPath", volume.Status.HostPath,
			)
			return err
		}
	}

	// Release volume from associated drive.
	if err := handler.releaseVolume(ctx, volume.Status.DriveName, volume.Name, volume.Status.TotalCapacity); err != nil {
		return err
	}

	volume.SetFinalizers(finalizers)
	_, err := client.VolumeClient().Update(
		ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)

	return err
}

func (handler *volumeEventHandler) releaseVolume(ctx context.Context, driveName, volumeName string, capacity int64) error {
	drive, err := client.DriveClient().Get(
		ctx, driveName, metav1.GetOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		return err
	}

	finalizers, found := excludeFinalizer(
		drive.GetFinalizers(), consts.DriveFinalizerPrefix+volumeName,
	)

	if found {
		drive.SetFinalizers(finalizers)
		drive.Status.FreeCapacity += capacity
		drive.Status.AllocatedCapacity = drive.Status.TotalCapacity - drive.Status.FreeCapacity

		_, err = client.DriveClient().Update(
			ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
	}

	return err
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
