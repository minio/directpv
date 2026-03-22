// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

type logWriter struct {
	buffer []byte
	closed bool
	mutex  sync.Mutex
}

func (w *logWriter) Write(data []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.closed {
		klog.Info(string(data))
		return len(data), io.EOF
	}

	w.buffer = append(w.buffer, data...)
	for {
		index := bytes.IndexRune(w.buffer, '\n')
		if index < 0 {
			break
		}

		klog.Info(string(w.buffer[:index+1]))
		w.buffer = w.buffer[index+1:]
	}

	return len(data), nil
}

func (w *logWriter) Close() (err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	klog.Info(string(w.buffer))
	w.buffer = nil
	w.closed = true
	return nil
}

func repair(ctx context.Context, drive *types.Drive, force, disablePrefetch, dryRun bool,
	getDeviceByFSUUID func(fsuuid string) (string, error),
	getMounts func() (mountInfo *sys.MountInfo, err error),
	unmount func(mountPoint string) error,
	repair func(ctx context.Context, device string, force, disablePrefetch, dryRun bool, output io.Writer) error,
	mount func(device, target string) (err error),
) error {
	device, err := getDeviceByFSUUID(drive.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				"on the host to reload",
			"FSUUID", drive.Status.FSUUID)
		client.Eventf(
			drive, client.EventTypeWarning, client.EventReasonStageVolume,
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", drive.Status.FSUUID)
		return fmt.Errorf("unable to find device by FSUUID %v; %w", drive.Status.FSUUID, err)
	}

	target := types.GetDriveMountDir(drive.Status.FSUUID)
	legacyTarget := path.Join(consts.LegacyAppRootDir, "mnt", drive.Status.FSUUID)

	mountInfo, err := getMounts()
	if err != nil {
		return err
	}

	mountPoints := make(utils.StringSet)
	for _, mountEntry := range mountInfo.FilterByMountSource(device).List() {
		switch mountEntry.MountPoint {
		case target, legacyTarget:
		default:
			mountPoints.Set(mountEntry.MountPoint)
		}
	}

	if len(mountPoints) != 0 {
		return fmt.Errorf("unable to run xfs repair; device %v still mounted in [%v]", device, strings.Join(mountPoints.ToSlice(), ","))
	}

	if err = unmount(target); err != nil {
		return err
	}

	if err = unmount(legacyTarget); err != nil {
		return err
	}

	logWriter := &logWriter{}
	err = repair(ctx, device, force, disablePrefetch, dryRun, logWriter)
	logWriter.Close()
	if err != nil {
		return err
	}

	merr := mount(device, target)
	if merr != nil {
		klog.ErrorS(merr, "unable to mount the drive", "Source", device, "Target", target)
	}

	driveID := drive.GetDriveID()
	updateFunc := func() error {
		drive, err := client.DriveClient().Get(ctx, string(driveID), metav1.GetOptions{})
		if err != nil {
			return err
		}

		if merr != nil {
			drive.SetMountErrorCondition(fmt.Sprintf("unable to mount; %v", merr))
			client.Eventf(drive,
				client.EventTypeWarning,
				client.EventReasonDriveMountError,
				"unable to mount the drive; %v", merr,
			)
			drive.Status.Status = directpvtypes.DriveStatusError
		} else {
			client.Eventf(
				drive,
				client.EventTypeNormal,
				client.EventReasonDriveMounted,
				"Drive mounted successfully to %s", target,
			)
			drive.Status.Status = directpvtypes.DriveStatusReady
		}

		_, err = client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
}

// Repair runs `xfs_repair` command on specified drive
func Repair(ctx context.Context, drive *types.Drive, force, disablePrefetch, dryRun bool) error {
	return repair(ctx, drive, force, disablePrefetch, dryRun,
		sys.GetDeviceByFSUUID,
		func() (mountInfo *sys.MountInfo, err error) {
			mountInfo, err = sys.NewMountInfo()
			return
		},
		func(mountPoint string) error {
			return sys.Unmount(mountPoint, true, true, false)
		},
		xfs.Repair,
		xfs.Mount,
	)
}
