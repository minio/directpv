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

type migrateTask struct {
	client       *client.Client
	legacyClient *legacyclient.Client
}

func (migrateTask) Name() string {
	return "Migration"
}

func (migrateTask) Start(ctx context.Context, args *Args) error {
	if !sendStartMessage(ctx, args.ProgressCh, 2) {
		return errSendProgress
	}
	return nil
}

func (migrateTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (t migrateTask) Execute(ctx context.Context, args *Args) error {
	return t.migrate(ctx, args, true)
}

func (migrateTask) Delete(_ context.Context, _ *Args) error {
	return nil
}

func (t migrateTask) migrateDrives(ctx context.Context, dryRun bool, progressCh chan<- Message) (driveMap map[string]string, legacyDriveErrors, driveErrors map[string]error, err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	driveMap = map[string]string{}
	legacyDriveErrors = map[string]error{}
	driveErrors = map[string]error{}

	fsUUIDs := make(utils.StringSet)
	for result := range t.legacyClient.ListDrives(ctx) {
		if result.Err != nil {
			return nil, legacyDriveErrors, driveErrors, fmt.Errorf(
				"unable to get legacy drives; %w", result.Err,
			)
		}

		if result.Drive.Status.DriveStatus != directv1beta5.DriveStatusReady &&
			result.Drive.Status.DriveStatus != directv1beta5.DriveStatusInUse {
			continue // ignore other than Ready/InUse drives
		}

		if !utils.IsUUID(result.Drive.Status.FilesystemUUID) {
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

		existingDrive, err := t.client.Drive().Get(ctx, string(driveID), metav1.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				if !dryRun {
					sendProgressMessage(ctx, progressCh, fmt.Sprintf("Migrating directcsidrive %s to directpvdrive %s", result.Drive.Name, drive.Name), 1, nil)
					_, err = t.client.Drive().Create(ctx, drive, metav1.CreateOptions{})
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

func (t migrateTask) migrateVolumes(ctx context.Context, driveMap map[string]string, dryRun bool, progressCh chan<- Message) (legacyVolumeErrors, volumeErrors map[string]error, err error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	legacyVolumeErrors = map[string]error{}
	volumeErrors = map[string]error{}

	for result := range t.legacyClient.ListVolumes(ctx) {
		if result.Err != nil {
			return legacyVolumeErrors, volumeErrors, fmt.Errorf(
				"unable to get legacy volumes; %w", result.Err,
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

		existingVolume, err := t.client.Volume().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			switch {
			case apierrors.IsNotFound(err):
				if !dryRun {
					sendProgressMessage(ctx, progressCh, fmt.Sprintf("Migrating directcsivolume %s to directpvvolume %s", result.Volume.Name, volume.Name), 2, nil)
					_, err = t.client.Volume().Create(ctx, volume, metav1.CreateOptions{})
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
func (t migrateTask) migrate(ctx context.Context, args *Args, installer bool) (err error) {
	if (installer && args.DryRun) || args.Declarative || !args.Legacy {
		return nil
	}

	legacyclient.Init()

	version, _, err := legacyclient.GetGroupVersion(t.legacyClient.K8sClient, "DirectCSIDrive")
	if err != nil {
		return fmt.Errorf("unable to probe DirectCSIDrive version; %w", err)
	}

	switch version {
	case directv1beta5.Version, directv1beta4.Version, directv1beta3.Version:
	default:
		return fmt.Errorf("migration does not support DirectCSIDrive version %v", version)
	}

	version, _, err = legacyclient.GetGroupVersion(t.legacyClient.K8sClient, "DirectCSIVolume")
	if err != nil {
		return fmt.Errorf("unable to probe DirectCSIVolume version; %w", err)
	}

	switch version {
	case directv1beta5.Version, directv1beta4.Version, directv1beta3.Version:
	default:
		return fmt.Errorf("migration does not support DirectCSIVolume version %v", version)
	}

	driveMap, legacyDriveErrors, driveErrors, err := t.migrateDrives(ctx, args.DryRun, args.ProgressCh)
	if err != nil {
		return err
	}
	if !sendProgressMessage(ctx, args.ProgressCh, "Migrated the drives", 1, nil) {
		return errSendProgress
	}
	legacyVolumeErrors, volumeErrors, err := t.migrateVolumes(ctx, driveMap, args.DryRun, args.ProgressCh)
	if err != nil {
		return err
	}
	if !sendProgressMessage(ctx, args.ProgressCh, "Migrated the volumes", 2, nil) {
		return errSendProgress
	}

	if len(legacyDriveErrors) != 0 {
		if err := migrateLog(ctx, args, fmt.Sprintf("legacy drive errors:\n%+v\n", legacyDriveErrors), false); err != nil {
			return err
		}
	}

	if len(legacyVolumeErrors) != 0 {
		if err := migrateLog(ctx, args, fmt.Sprintf("legacy volume errors:\n%+v\n", legacyVolumeErrors), false); err != nil {
			return err
		}
	}

	if len(driveErrors) != 0 {
		if err := migrateLog(ctx, args, fmt.Sprintf("drive errors:\n%+v\n", driveErrors), true); err != nil {
			return err
		}
	}

	if len(volumeErrors) != 0 {
		if err := migrateLog(ctx, args, fmt.Sprintf("volume errors:\n%+v\n", volumeErrors), true); err != nil {
			return err
		}
	}

	return nil
}

// Migrate migrates the resources using the provided clients
func Migrate(ctx context.Context, args *Args, client *client.Client, legacyClient *legacyclient.Client) error {
	return migrateTask{client, legacyClient}.migrate(ctx, args, false)
}
