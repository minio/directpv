// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package device

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

type device struct {
	Device
	FSUUID        string
	Label         string
	TotalCapacity int64
	FreeCapacity  int64
}

func probeDeviceMap() (map[string][]device, error) {
	devices, err := Probe()
	if err != nil {
		return nil, err
	}

	deviceMap := map[string][]device{}
	for _, dev := range devices {
		if dev.Hidden || dev.Partitioned || len(dev.Holders) != 0 || dev.SwapOn || dev.CDROM || dev.Size == 0 {
			continue
		}

		fsuuid, label, totalCapacity, freeCapacity, err := xfs.Probe(utils.AddDevPrefix(dev.Name))
		if err != nil {
			if !errors.Is(err, xfs.ErrFSNotFound) {
				klog.ErrorS(err, "unable to probe XFS filesystem", "Device", dev.Name)
			}
			continue
		}

		deviceMap[fsuuid] = append(deviceMap[fsuuid], device{
			Device:        dev,
			FSUUID:        fsuuid,
			Label:         label,
			TotalCapacity: int64(totalCapacity),
			FreeCapacity:  int64(freeCapacity),
		})
	}

	return deviceMap, nil
}

func syncDrive(drive *types.Drive, device device) (updated bool) {
	if string(drive.GetDriveName()) != device.Name {
		updated = true
		drive.SetDriveName(directpvtypes.DriveName(device.Name))
	}
	if drive.Status.TotalCapacity != device.TotalCapacity {
		updated = true
		drive.Status.TotalCapacity = device.TotalCapacity
		drive.Status.FreeCapacity = drive.Status.TotalCapacity - drive.Status.AllocatedCapacity
		if drive.Status.FreeCapacity < 0 {
			drive.Status.FreeCapacity = 0
		}
	}
	if drive.Status.Make != device.Make() {
		updated = true
		drive.Status.Make = device.Make()
	}
	return
}

func verifyDrive(drive *types.Drive) (updated bool) {
	switch drive.Status.Status {
	case directpvtypes.DriveStatusReady, directpvtypes.DriveStatusLost, directpvtypes.DriveStatusError, directpvtypes.DriveStatusMoving:
	default:
		return false
	}

	source := utils.AddDevPrefix(string(drive.GetDriveName()))
	target := types.GetDriveMountDir(drive.Status.FSUUID)
	if err := xfs.Mount(source, target); err != nil {
		drive.Status.Status = directpvtypes.DriveStatusError
		drive.SetMountErrorCondition(fmt.Sprintf("unable to mount; %v", err))
		client.Eventf(drive, client.EventTypeWarning, client.EventReasonDriveMountError, "unable to mount the drive; %v", err)
		klog.ErrorS(err, "unable to mount the drive", "Source", source, "Target", target)
		return true
	}

	client.Eventf(
		drive,
		client.EventTypeNormal,
		client.EventReasonDriveMounted,
		"Drive mounted successfully to %s", target,
	)

	if drive.Status.Status == directpvtypes.DriveStatusLost {
		updated = true
		drive.Status.Status = directpvtypes.DriveStatusReady
	}

	latestErrorConditionType := drive.GetLatestErrorConditionType()
	if drive.Status.Status == directpvtypes.DriveStatusError {
		switch latestErrorConditionType {
		case directpvtypes.DriveConditionTypeMountError, directpvtypes.DriveConditionTypeMultipleMatches, directpvtypes.DriveConditionTypeRelabelError:
			updated = true
			drive.Status.Status = directpvtypes.DriveStatusReady
		}
	}

	if !drive.Spec.Relabel {
		return updated
	}

	err := os.Symlink(".", types.GetVolumeRootDir(drive.Status.FSUUID))
	if err != nil {
		switch {
		case errors.Is(err, os.ErrExist):
			err = nil
		default:
			if latestErrorConditionType != directpvtypes.DriveConditionTypeIOError {
				drive.Status.Status = directpvtypes.DriveStatusError
				drive.SetRelabelErrorCondition(fmt.Sprintf("unable to relabel; %v", err))
			}

			client.Eventf(
				drive,
				client.EventTypeWarning,
				client.EventReasonDriveRelabelError,
				"unable to relabel; %v", err,
			)

			klog.ErrorS(
				err,
				"unable to create symlink",
				"symlink", types.GetVolumeRootDir(drive.Status.FSUUID),
				"drive", drive.Name,
			)
		}
	}

	if err == nil {
		drive.Spec.Relabel = false
	}

	return true
}

func updateDrive(ctx context.Context, driveName string, deviceMap map[string][]device, mountInfo *sys.MountInfo) error {
	drive, err := client.DriveClient().Get(ctx, driveName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	var updated bool
	devices := deviceMap[drive.Status.FSUUID]
	switch len(devices) {
	case 0:
		// no match
		if drive.Status.Status != directpvtypes.DriveStatusLost {
			updated = true
			drive.Status.Status = directpvtypes.DriveStatusLost
		}
	case 1:
		// device found

		mountPoint := types.GetDriveMountDir(drive.Status.FSUUID)
		legacyMountPoint := path.Join(consts.LegacyAppRootDir, "mnt", drive.Status.FSUUID)

		mountPoints := make(utils.StringSet)
		for _, mountEntry := range mountInfo.FilterByMountSource(devices[0].MajorMinor).FilterByRoot("/").List() {
			switch mountEntry.MountPoint {
			case mountPoint, legacyMountPoint:
			default:
				mountPoints.Set(mountEntry.MountPoint)
			}

			check := func() error {
				switch {
				case mountEntry.FilesystemType != "xfs":
					return errors.New("device filesystem is not XFS")
				case !mountEntry.MountOptions.Exist("prjquota"):
					return errors.New("device mounted without 'prjquota' mount option")
				case !mountEntry.MountOptions.Exist("noatime"):
					return errors.New("device mounted without 'noatime' mount option")
				case !mountEntry.MountOptions.Exist("ro"):
					return errors.New("device mounted in read-only mode")
				default:
					return nil
				}
			}

			if err := check(); err != nil {
				klog.ErrorS(err, "device", mountEntry.MountSource, "filesystemType", mountEntry.FilesystemType, "drive", drive.GetDriveID(), drive.GetDriveName())
			}
		}

		if len(mountPoints) != 0 {
			klog.ErrorS(errors.New("device mounted outside of DirectPV"), "device", devices[0].Name, "mountPoints", mountPoints.ToSlice(), "drive", drive.GetDriveID(), drive.GetDriveName())
		}

		updated = syncDrive(drive, devices[0])
		if verifyDrive(drive) {
			updated = true
		}
	default:
		// more than one matching devices
		updated = true
		var deviceNames string
		for i := range devices {
			if deviceNames != "" {
				deviceNames += ", "
			}
			deviceNames += devices[i].Name
		}
		drive.Status.Status = directpvtypes.DriveStatusError
		drive.SetMultipleMatchesErrorCondition(fmt.Sprintf("multiple devices found by FSUUID; %v", deviceNames))
		client.Eventf(drive, client.EventTypeWarning, client.EventReasonDriveHasMultipleMatches, "unable to mount the drive due to %v", err)
		klog.ErrorS(err, "multiple devices found by FSUUID", "drive", drive.GetDriveName(), "FSUUID", drive.Status.FSUUID, "devices", deviceNames)
	}
	if updated {
		_, err = client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
	}
	return err
}

// Sync - matches and syncs the drive with locally probed device
func Sync(ctx context.Context, nodeID directpvtypes.NodeID) error {
	deviceMap, err := probeDeviceMap()
	if err != nil {
		return err
	}

	mountInfo, err := sys.NewMountInfo()
	if err != nil {
		return err
	}

	drives, err := client.NewDriveLister().NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(nodeID))}).Get(ctx)
	if err != nil {
		return err
	}
	for i := range drives {
		if err = retry.RetryOnConflict(retry.DefaultRetry, func() error { return updateDrive(ctx, drives[i].Name, deviceMap, mountInfo) }); err != nil {
			return err
		}
	}

	return nil
}
