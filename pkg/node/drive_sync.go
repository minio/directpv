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

package node

import (
	"context"
	"strings"

	"github.com/google/uuid"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

func syncDrives(ctx context.Context, nodeID string, loopbackOnly bool, topology map[string]string) error {
	devices, err := sys.ProbeDevices()
	if err != nil {
		klog.ErrorS(err, "unable to probe devices")
		return err
	}

	resultCh, err := client.ListDrives(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(nodeID)},
		nil,
		nil,
		client.MaxThreadCount,
	)
	if err != nil {
		klog.Error(err)
		return err
	}

	for result := range resultCh {
		if result.Err != nil {
			klog.Error(result.Err)
			return result.Err
		}

		if matched, updated := updateDrive(ctx, &result.Drive, devices); matched {
			switch result.Drive.Status.DriveStatus {
			case directcsi.DriveStatusReady, directcsi.DriveStatusInUse:
				if updated {
					mountDrive(ctx, &result.Drive)
				}
			}
		} else {
			if err := client.DeleteDrive(ctx, &result.Drive, true); err != nil {
				klog.ErrorS(err, "unable to delete drive", "Name", result.Drive.Name, "Status.Path", result.Drive.Status.Path)
			}
		}
	}

	for _, device := range devices {
		if !loopbackOnly && sys.LoopRegexp.MatchString(device.Name) {
			klog.V(5).InfoS("loopback device is ignored", "Name", device.Name)
			continue
		}

		drive := client.NewDirectCSIDrive(
			uuid.New().String(),
			client.NewDirectCSIDriveStatus(device, nodeID, topology),
		)

		if err := client.CreateDrive(ctx, drive); err != nil {
			klog.ErrorS(err, "unable to create drive", "Status.Path", drive.Status.Path)
		}
	}

	return nil
}

func syncDrive(
	ctx context.Context,
	devices map[string]*sys.Device,
	drive *directcsi.DirectCSIDrive,
	matchFunc func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool,
	matchName string,
) (matched bool, CRDUpdated bool) {
	for _, device := range devices {
		if !matchFunc(drive, device) {
			// This device and drive do not match by properties WRT match function.
			// Try next device.
			continue
		}

		delete(devices, device.Name)

		var updated, nameChanged bool
		CRDUpdated = true
		if updated, nameChanged = updateDriveProperties(drive, device); updated {
			_, err := client.GetLatestDirectCSIDriveInterface().Update(
				ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
			)
			if err != nil {
				klog.ErrorS(err, "unable to update drive by "+matchName, "Path", drive.Status.Path, "device.Name", device.Name)
				CRDUpdated = false
			}

			if err == nil && nameChanged {
				volumeInterface := client.GetLatestDirectCSIVolumeInterface()

				updateLabels := func(volumeName, driveName string) func() error {
					return func() error {
						volume, err := volumeInterface.Get(
							ctx, volumeName, metav1.GetOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
						)
						if err != nil {
							return err
						}

						volume.Labels[string(utils.DrivePathLabelKey)] = driveName
						_, err = volumeInterface.Update(
							ctx, volume, metav1.UpdateOptions{TypeMeta: utils.DirectCSIVolumeTypeMeta()},
						)
						return err
					}
				}

				for _, finalizer := range drive.GetFinalizers() {
					if !strings.HasPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix) {
						continue
					}

					volumeName := strings.TrimPrefix(finalizer, directcsi.DirectCSIDriveFinalizerPrefix)
					go func() {
						err := retry.RetryOnConflict(retry.DefaultRetry, updateLabels(volumeName, utils.SanitizeDrivePath(drive.Status.Path)))
						if err != nil {
							klog.ErrorS(err, "unable to update volume %v", volumeName)
						}
					}()
				}
			}
		}

		return true, CRDUpdated
	}

	// None of devices match.
	return false, false
}
