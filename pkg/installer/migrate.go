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

package installer

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	directv1beta3 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta3"
	directv1beta4 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta4"
	directv1beta5 "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	legacyAccessTierLabelKey = legacyclient.GroupName + "/access-tier"
	legacyCreatedByLabelKey  = legacyclient.GroupName + "/created-by"
	legacyNodeLabelKey       = legacyclient.GroupName + "/node"
	legacyPathLabelKey       = legacyclient.GroupName + "/path"
	legacyVersionLabelKey    = legacyclient.GroupName + "/version"
	legacyDriveLabelKey      = legacyclient.GroupName + "/drive"
	legacyDrivePathLabelKey  = legacyclient.GroupName + "/drive-path"
	legacyPVProtection       = legacyclient.GroupName + "/pv-protection"
	legacyPurgeProtection    = legacyclient.GroupName + "/purge-protection"
)

var uuidRegex = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")

func migrateDrives(ctx context.Context, dryRun bool) (driveMap map[string]string, legacyDriveErrors map[string]error, driveErrors map[string]error, err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	driveMap = map[string]string{}
	legacyDriveErrors = map[string]error{}
	driveErrors = map[string]error{}

	fsUUIDs := make(utils.StringSet)
	for result := range legacyclient.ListDrives(ctx) {
		if result.Err != nil {
			return nil, legacyDriveErrors, driveErrors, fmt.Errorf(
				"unable to get legacy drives; %v", result.Err,
			)
		}

		if result.Drive.Status.DriveStatus != directv1beta5.DriveStatusReady &&
			result.Drive.Status.DriveStatus != directv1beta5.DriveStatusInUse {
			continue // ignore other than Ready/InUse drives
		}

		if !uuidRegex.MatchString(result.Drive.Status.FilesystemUUID) {
			legacyDriveErrors[result.Drive.Name] = fmt.Errorf(
				"invalid filesystem UUID %v in legacy drive %v",
				result.Drive.Status.FilesystemUUID, result.Drive.Name,
			)
			continue
		}

		if fsUUIDs.Exist(result.Drive.Status.FilesystemUUID) {
			legacyDriveErrors[result.Drive.Name] = fmt.Errorf(
				"duplicate filesystem UUID %v found in legacy drive %v",
				result.Drive.Status.FilesystemUUID, result.Drive.Name,
			)
			continue
		}

		fsUUIDs.Set(result.Drive.Status.FilesystemUUID)
		driveMap[result.Drive.Name] = result.Drive.Status.FilesystemUUID

		driveID := directpvtypes.DriveID(result.Drive.Status.FilesystemUUID)
		nodeID := directpvtypes.NodeID(result.Drive.Status.NodeName)
		driveName := directpvtypes.DriveName(utils.TrimDevPrefix(result.Drive.Status.Path))
		accessTier := directpvtypes.AccessTierDefault
		switch result.Drive.Status.AccessTier {
		case directv1beta5.AccessTierCold, directv1beta5.AccessTierWarm, directv1beta5.AccessTierHot:
			accessTier = directpvtypes.AccessTier(result.Drive.Status.AccessTier)
		}

		topology := map[string]string{}
		for key, value := range result.Drive.Status.Topology {
			if strings.HasPrefix(key, legacyclient.GroupName) {
				key = strings.Replace(key, legacyclient.GroupName, consts.GroupName, 1)
			}

			if key == string(directpvtypes.TopologyDriverIdentity) &&
				strings.HasPrefix(value, legacyclient.Identity) {
				value = strings.Replace(value, legacyclient.Identity, consts.Identity, 1)
			}
			topology[key] = value
		}

		status := types.DriveStatus{
			TotalCapacity:     result.Drive.Status.TotalCapacity,
			AllocatedCapacity: result.Drive.Status.AllocatedCapacity,
			FreeCapacity:      result.Drive.Status.FreeCapacity,
			FSUUID:            result.Drive.Status.FilesystemUUID,
			Status:            directpvtypes.DriveStatusReady,
			Topology:          topology,
		}

		drive := types.NewDrive(
			driveID,
			status,
			nodeID,
			driveName,
			accessTier,
		)
		drive.SetMigratedLabel()
		drive.Spec.Relabel = true

		for key, value := range result.Drive.Labels {
			switch key {
			case legacyAccessTierLabelKey:
			case legacyCreatedByLabelKey:
			case legacyNodeLabelKey:
			case legacyPathLabelKey:
			case legacyVersionLabelKey:
			default:
				if strings.HasPrefix(key, legacyclient.GroupName) {
					key = strings.Replace(key, legacyclient.GroupName, consts.GroupName, 1)
				}
				drive.Labels[key] = value
			}
		}

		for _, finalizer := range result.Drive.Finalizers {
			if strings.HasPrefix(finalizer, legacyclient.GroupName) {
				finalizer = strings.Replace(finalizer, legacyclient.GroupName, consts.GroupName, 1)
			}

			if !utils.Contains(drive.Finalizers, finalizer) {
				drive.Finalizers = append(drive.Finalizers, finalizer)
			}
		}

		existingDrive, err := client.DriveClient().Get(ctx, string(driveID), metav1.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				if !dryRun {
					_, err = client.DriveClient().Create(ctx, drive, metav1.CreateOptions{})
					if err != nil {
						legacyDriveErrors[result.Drive.Name] = fmt.Errorf(
							"unable to create drive %v by migrating legacy drive %v; %w",
							driveID, result.Drive.Name, err,
						)
					}
				}
			default:
				driveErrors[string(driveID)] = fmt.Errorf(
					"unable to get drive by drive ID %v; %w", driveID, err,
				)
				delete(driveMap, result.Drive.Name)
			}
		} else {
			if existingDrive.IsMigrated() {
				legacyDriveErrors[result.Drive.Name] = fmt.Errorf(
					"legacy drive %v is already migrated to drive %v",
					result.Drive.Name, existingDrive.Name,
				)
			} else {
				legacyDriveErrors[result.Drive.Name] = fmt.Errorf(
					"unable to migrate legacy drive %v; drive %v already exists",
					result.Drive.Name, existingDrive.Name,
				)
			}
		}
	}

	return driveMap, legacyDriveErrors, driveErrors, nil
}

func migrateVolumes(ctx context.Context, driveMap map[string]string, dryRun bool) (legacyVolumeErrors map[string]error, volumeErrors map[string]error, err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	legacyVolumeErrors = map[string]error{}
	volumeErrors = map[string]error{}

	for result := range legacyclient.ListVolumes(ctx) {
		if result.Err != nil {
			return legacyVolumeErrors, volumeErrors, fmt.Errorf(
				"unable to get legacy volumes; %v", result.Err,
			)
		}

		fsuuid, found := driveMap[result.Volume.Status.Drive]
		if !found {
			legacyVolumeErrors[result.Volume.Name] = fmt.Errorf(
				"referring drive %v of volume %v not found",
				result.Volume.Status.Drive, result.Volume.Name,
			)
			continue
		}

		name := result.Volume.Name
		nodeID := directpvtypes.NodeID(result.Volume.Status.NodeName)
		driveID := directpvtypes.DriveID(fsuuid)
		driveName := directpvtypes.DriveName(result.Volume.Labels["direct.csi.min.io/drive-path"])
		size := result.Volume.Status.TotalCapacity

		volume := types.NewVolume(
			name,
			fsuuid,
			nodeID,
			driveID,
			driveName,
			size,
		)
		volume.SetMigratedLabel()
		volume.Status.DataPath = result.Volume.Status.HostPath
		volume.Status.StagingTargetPath = result.Volume.Status.StagingPath
		volume.Status.TargetPath = result.Volume.Status.ContainerPath
		volume.Status.AvailableCapacity = result.Volume.Status.AvailableCapacity
		volume.Status.UsedCapacity = result.Volume.Status.UsedCapacity
		if volume.Status.StagingTargetPath != "" {
			volume.Status.Status = directpvtypes.VolumeStatusReady
		}

		for key, value := range result.Volume.Labels {
			switch key {
			case legacyCreatedByLabelKey:
			case legacyDriveLabelKey:
			case legacyDrivePathLabelKey:
			case legacyNodeLabelKey:
			case legacyVersionLabelKey:
			default:
				if strings.HasPrefix(key, legacyclient.GroupName) {
					key = strings.Replace(key, legacyclient.GroupName, consts.GroupName, 1)
				}
				volume.Labels[key] = value
			}
		}

		for _, finalizer := range result.Volume.Finalizers {
			switch finalizer {
			case legacyPVProtection:
			case legacyPurgeProtection:
			default:
				if strings.HasPrefix(finalizer, legacyclient.GroupName) {
					finalizer = strings.Replace(finalizer, legacyclient.GroupName, consts.GroupName, 1)
				}

				if !utils.Contains(volume.Finalizers, finalizer) {
					volume.Finalizers = append(volume.Finalizers, finalizer)
				}
			}
		}

		existingVolume, err := client.VolumeClient().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				if !dryRun {
					_, err = client.VolumeClient().Create(ctx, volume, metav1.CreateOptions{})
					if err != nil {
						legacyVolumeErrors[result.Volume.Name] = fmt.Errorf(
							"unable to create volume %v by migrating legacy volume %v; %w",
							name, result.Volume.Name, err,
						)
					}
				}
			default:
				volumeErrors[name] = fmt.Errorf("unable to get volume %v; %w", name, err)
			}
		} else {
			if existingVolume.IsMigrated() {
				legacyVolumeErrors[result.Volume.Name] = fmt.Errorf(
					"legacy volume %v is already migrated to volume %v",
					result.Volume.Name, existingVolume.Name,
				)
			} else {
				legacyVolumeErrors[result.Volume.Name] = fmt.Errorf(
					"unable to migrate legacy volume %v; volume %v already exists",
					result.Volume.Name, existingVolume.Name,
				)
			}
		}
	}

	return legacyVolumeErrors, volumeErrors, nil
}

// Migrate migrates legacy drives and volumes.
func Migrate(ctx context.Context, dryRun bool) error {
	legacyclient.Init()

	version, _, err := legacyclient.GetGroupVersion("DirectCSIDrive")
	if err != nil {
		return fmt.Errorf("unable to probe DirectCSIDrive version; %w", err)
	}

	switch version {
	case directv1beta5.Version, directv1beta4.Version, directv1beta3.Version:
	default:
		return fmt.Errorf("migration does not support DirectCSIDrive version %v", version)
	}

	version, _, err = legacyclient.GetGroupVersion("DirectCSIVolume")
	if err != nil {
		return fmt.Errorf("unable to probe DirectCSIVolume version; %w", err)
	}

	switch version {
	case directv1beta5.Version, directv1beta4.Version, directv1beta3.Version:
	default:
		return fmt.Errorf("migration does not support DirectCSIVolume version %v", version)
	}

	driveMap, legacyDriveErrors, driveErrors, err := migrateDrives(ctx, dryRun)
	if err != nil {
		return err
	}

	legacyVolumeErrors, volumeErrors, err := migrateVolumes(ctx, driveMap, dryRun)
	if err != nil {
		return err
	}

	if len(legacyDriveErrors) != 0 {
		fmt.Printf("legacy drive errors:\n%+v\n", legacyDriveErrors)
	}

	if len(driveErrors) != 0 {
		fmt.Printf("drive errors:\n%+v\n", driveErrors)
	}

	if len(legacyVolumeErrors) != 0 {
		fmt.Printf("legacy volume errors:\n%+v\n", legacyVolumeErrors)
	}

	if len(volumeErrors) != 0 {
		fmt.Printf("volume errors:\n%+v\n", volumeErrors)
	}

	return nil
}
