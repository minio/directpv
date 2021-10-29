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

package node

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/uevent"
	"github.com/minio/direct-csi/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

type ueventHandler struct {
	listener        *uevent.Listener
	nodeID          string
	topology        map[string]string
	directCSIClient clientset.Interface
	syncMu          sync.Mutex
}

func startUeventHandler(ctx context.Context, nodeID string, topology map[string]string) {
	klog.V(3).Info("Starting uevent handler")
	handler := &ueventHandler{
		nodeID:          nodeID,
		directCSIClient: utils.GetDirectClientset(),
		topology:        topology,
	}
	handler.processLoop(ctx)
}

func (handler *ueventHandler) syncDrive(
	ctx context.Context,
	devices map[string]*sys.Device,
	drive directcsi.DirectCSIDrive,
	matchFunc func(drive directcsi.DirectCSIDrive, device *sys.Device) bool,
	matchName string,
) bool {
	for _, device := range devices {
		if !matchFunc(drive, device) {
			// This device and drive do not match by properties WRT match function.
			// Try next device.
			continue
		}

		delete(devices, device.Name)

		var updated, nameChanged bool
		if drive, updated, nameChanged = updateDriveProperties(drive, device); updated {
			_, err := handler.directCSIClient.DirectV1beta3().DirectCSIDrives().Update(
				ctx, &drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
			)
			if err != nil {
				klog.ErrorS(err, "unable to update drive by "+matchName, "Path", drive.Status.Path, "device.Name", device.Name)
			}

			if err == nil && nameChanged {
				volumeInterface := handler.directCSIClient.DirectV1beta3().DirectCSIVolumes()

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

		return true
	}

	// None of devices match.
	return false
}

func (handler *ueventHandler) updateDrive(ctx context.Context, drive directcsi.DirectCSIDrive, devices map[string]*sys.Device) bool {
	switch {
	case isHWInfoAvailable(drive):
		return handler.syncDrive(ctx, devices, drive, matchDeviceHWInfo, "hardware IDs")
	case isDMMDUUIDAvailable(drive):
		return handler.syncDrive(ctx, devices, drive, matchDeviceDMMDUUID, "DM/MD UUIDs")
	case isPTUUIDAvailable(drive):
		return handler.syncDrive(ctx, devices, drive, matchDevicePTUUID, "Partition Table UUID")
	case isPartUUIDAvailable(drive):
		return handler.syncDrive(ctx, devices, drive, matchDevicePartUUID, "Partition UUID")
	case isFSUUIDAvailable(drive):
		return handler.syncDrive(ctx, devices, drive, matchDeviceFSUUID, "Fileystem UUIDs")
	default:
		return false
	}
}

func (handler *ueventHandler) syncDrives(ctx context.Context) {
	handler.syncMu.Lock()
	defer handler.syncMu.Unlock()

	devices, err := sys.ProbeDevices()
	if err != nil {
		klog.ErrorS(err, "unable to probe drives")
		return
	}

	resultCh, err := utils.ListDrives(
		ctx,
		handler.directCSIClient.DirectV1beta3().DirectCSIDrives(),
		[]utils.LabelValue{utils.NewLabelValue(handler.nodeID)},
		nil,
		nil,
		utils.MaxThreadCount,
	)
	if err != nil {
		klog.Error(err)
		return
	}

	for result := range resultCh {
		if result.Err != nil {
			klog.Error(result.Err)
			return
		}

		if !handler.updateDrive(ctx, result.Drive, devices) {
			if err := utils.DeleteDrive(ctx, handler.directCSIClient.DirectV1beta3().DirectCSIDrives(), &result.Drive, true); err != nil {
				klog.ErrorS(err, "unable to delete drive", "Name", result.Drive.Name, "Status.Path", result.Drive.Status.Path)
			}
		}
	}

	for _, device := range devices {
		drive := utils.NewDirectCSIDrive(
			uuid.New().String(),
			utils.NewDirectCSIDriveStatus(device, handler.nodeID, handler.topology),
		)
		if err := utils.CreateDrive(ctx, handler.directCSIClient.DirectV1beta3().DirectCSIDrives(), drive); err != nil {
			klog.ErrorS(err, "unable to create drive", "Status.Path", drive.Status.Path)
		}
	}
}

func (handler *ueventHandler) removeDrive(ctx context.Context, drive directcsi.DirectCSIDrive, devices map[string]*sys.Device) {
	for _, device := range devices {
		remove := func(matchFunc func(drive directcsi.DirectCSIDrive, device *sys.Device) bool) {
			if !matchFunc(drive, device) {
				// This device and drive do not match by properties WRT match function.
				// Try next device.
				return
			}

			delete(devices, device.Name)

			_, err := os.Stat(drive.Status.Path)
			switch {
			case err == nil:
				klog.ErrorS(os.ErrExist, "unable to delete drive", "Name", drive.Name, "Status.Path", drive.Status.Path)
			case !errors.Is(err, os.ErrNotExist):
				klog.ErrorS(err, "unable to delete drive", "Name", drive.Name, "Status.Path", drive.Status.Path)
			default:
				if err := utils.DeleteDrive(ctx, handler.directCSIClient.DirectV1beta3().DirectCSIDrives(), &drive, true); err != nil {
					klog.ErrorS(err, "unable to delete drive", "Name", drive.Name, "Status.Path", drive.Status.Path)
				}
			}
		}

		switch {
		case isHWInfoAvailable(drive):
			remove(matchDeviceHWInfo)
		case isDMMDUUIDAvailable(drive):
			remove(matchDeviceDMMDUUID)
		case isPTUUIDAvailable(drive):
			remove(matchDevicePTUUID)
		case isPartUUIDAvailable(drive):
			remove(matchDevicePartUUID)
		case isFSUUIDAvailable(drive):
			remove(matchDeviceFSUUID)
		default:
			remove(func(drive directcsi.DirectCSIDrive, device *sys.Device) bool {
				return drive.Status.Path == "/dev/"+device.Name && drive.Status.DriveStatus != directcsi.DriveStatusInUse
			})
		}
	}
}

func (handler *ueventHandler) processEvent(ctx context.Context, device *sys.Device, action string) {
	handler.syncMu.Lock()
	defer handler.syncMu.Unlock()

	resultCh, err := utils.ListDrives(
		ctx,
		handler.directCSIClient.DirectV1beta3().DirectCSIDrives(),
		[]utils.LabelValue{utils.NewLabelValue(handler.nodeID)},
		[]utils.LabelValue{utils.NewLabelValue(device.Name)},
		nil,
		utils.MaxThreadCount,
	)
	if err != nil {
		klog.Error(err)
		return
	}

	devices := map[string]*sys.Device{device.Name: device}

	for result := range resultCh {
		if result.Err != nil {
			klog.Error(err)
			return
		}

		drive := result.Drive

		if action == uevent.Remove {
			handler.removeDrive(ctx, drive, devices)
		} else {
			handler.updateDrive(ctx, drive, devices)
		}

		if len(devices) == 0 {
			return
		}
	}

	if action == uevent.Remove {
		klog.Errorf("no drives found to remove by IDs for device %v", device.Name)
		return
	}

	drive := utils.NewDirectCSIDrive(
		uuid.New().String(),
		utils.NewDirectCSIDriveStatus(device, handler.nodeID, handler.topology),
	)
	if err := utils.CreateDrive(ctx, handler.directCSIClient.DirectV1beta3().DirectCSIDrives(), drive); err != nil {
		klog.ErrorS(err, "unable to create drive", "Status.Path", drive.Status.Path)
	}
}

func (handler *ueventHandler) startListener(ctx context.Context) (err error) {
	backoff := &wait.Backoff{
		Steps:    4,
		Duration: 10 * time.Second,
		Factor:   5.0,
		Jitter:   0.1,
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		if handler.listener, err = uevent.StartListener(); err == nil {
			return
		}

		klog.Error(err)
		ticker.Reset(backoff.Step())
		select {
		case <-ctx.Done():
			return errors.New("cancelled by context")
		case <-ticker.C:
		}
	}
}

func (handler *ueventHandler) get(ctx context.Context) (map[string]string, error) {
	for {
		if handler.listener == nil {
			if err := handler.startListener(ctx); err != nil {
				return nil, err
			}
		}

		event, err := handler.listener.Get(ctx)
		if err == nil {
			return event, nil
		}

		klog.Error(err)
		handler.listener.Close()
		if err = handler.startListener(ctx); err != nil {
			return nil, err
		}
	}
}

func (handler *ueventHandler) processLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handler.syncDrives(ctx)
			}
		}
	}()

	for {
		event, err := handler.get(ctx)
		if err != nil {
			klog.Error(err)
			return
		}

		if sys.LoopRegexp.MatchString(path.Base(event["DEVPATH"])) {
			klog.V(5).InfoS("loopback device is ignored", "ACTION", event["ACTION"], "DEVPATH", event["DEVPATH"])
			continue
		}

		device, err := sys.CreateDevice(event)
		if err != nil {
			klog.ErrorS(err, "ACTION", event["ACTION"], "DEVPATH", event["DEVPATH"])
			continue
		}

		handler.processEvent(ctx, device, event["ACTION"])
	}
}
