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

package drive

import (
	"context"
	"errors"
	"fmt"
	"os"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/listener"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

var (
	errDriveNotEmpty = errors.New("drive is not empty")
)

type driveEventHandler struct {
	nodeID             string
	getDeviceByFSUUID  func(fsuuid string) (string, error)
	unmountDriveMounts func(devicePath string, force, detach, expire bool) error
}

func newDriveEventHandler(nodeID string) *driveEventHandler {
	return &driveEventHandler{
		nodeID:             nodeID,
		getDeviceByFSUUID:  device.GetDeviceByFSUUID,
		unmountDriveMounts: sys.UnmountDriveMounts,
	}
}

func (handler *driveEventHandler) ListerWatcher() cache.ListerWatcher {
	return DrivesListerWatcher(handler.nodeID)
}

func (handler *driveEventHandler) Name() string {
	return "drive"
}

func (handler *driveEventHandler) ObjectType() runtime.Object {
	return &types.Drive{}
}

func (handler *driveEventHandler) Handle(ctx context.Context, args listener.EventArgs) error {
	switch args.Event {
	case listener.AddEvent, listener.UpdateEvent:
		return handler.handleUpdate(ctx, args.Object.(*types.Drive))
	case listener.DeleteEvent:
	}
	return nil
}

func (handler *driveEventHandler) handleUpdate(ctx context.Context, drive *types.Drive) error {
	if drive.Status.Status == directpvtypes.DriveStatusReleased {
		return handler.release(ctx, drive)
	}
	return nil
}

func (handler *driveEventHandler) release(ctx context.Context, drive *types.Drive) error {
	if err := handler.verifyDriveState(ctx, drive); err != nil {
		return err
	}
	finalizers := drive.GetFinalizers()
	if len(finalizers) > 1 {
		return fmt.Errorf("unable to release drive %s. the drive still has volumes to be cleaned up", drive.Name)
	}
	if err := handler.unmountDriveMounts(drive.Status.Path, true, true, false); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			return err
		}
	}
	drive.Finalizers, _ = utils.ExcludeFinalizer(
		finalizers, consts.DriveFinalizerDataProtection,
	)
	if _, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()}); err != nil {
		return err
	}
	return client.DriveClient().Delete(ctx, drive.Name, metav1.DeleteOptions{})
}

func (handler *driveEventHandler) verifyDriveState(ctx context.Context, drive *types.Drive) error {
	device, err := handler.getDeviceByFSUUID(drive.Status.FSUUID)
	if err != nil {
		return err
	}
	if drive.Status.Path != device {
		return fmt.Errorf("path mismatch. expected: %s found: %s", drive.Status.Path, device)
	}
	return nil
}

// StartController starts drive controller.
func StartController(ctx context.Context, nodeID string) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	listener := listener.NewListener(newDriveEventHandler(nodeID), "drive-controller", hostname, 40)
	return listener.Run(ctx)
}
