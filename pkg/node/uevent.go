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
	"errors"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/uevent"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

func mountDrive(ctx context.Context, drive *directcsi.DirectCSIDrive) {
	target := filepath.Join(sys.MountRoot, drive.Status.FilesystemUUID)
	var flags []string
	if drive.Spec.RequestedFormat != nil {
		flags = drive.Spec.RequestedFormat.MountOptions
	}
	err := mount.MountXFSDevice(drive.Status.Path, target, flags)
	if err == nil {
		return
	}

	klog.ErrorS(err, "unable to mount drive", "Status.Path", drive.Status.Path, "Target", target, "Flags", flags)
	utils.UpdateCondition(
		drive.Status.Conditions,
		string(directcsi.DirectCSIDriveConditionInitialized),
		utils.BoolToCondition(false),
		string(directcsi.DirectCSIDriveReasonInitialized),
		err.Error(),
	)
	driveInterface := client.GetLatestDirectCSIDriveInterface()
	err = retry.RetryOnConflict(
		retry.DefaultRetry,
		func() (err error) {
			_, err = driveInterface.Update(ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()})
			return err
		},
	)
	if err != nil {
		klog.ErrorS(err, "unable to update drive", "Name", drive.Name, "Path", drive.Status.Path)
	}
}

type ueventHandler struct {
	listener              *uevent.Listener
	nodeID                string
	topology              map[string]string
	dynamicDriveDiscovery bool
	loopbackOnly          bool
	syncMu                sync.Mutex
}

func (handler *ueventHandler) syncDrives(ctx context.Context) {
	handler.syncMu.Lock()
	defer handler.syncMu.Unlock()
	if err := syncDrives(ctx, handler.nodeID, handler.loopbackOnly, handler.topology); err != nil {
		klog.ErrorS(err, "unable to sync drives")
	}
}

func updateDrive(ctx context.Context, drive *directcsi.DirectCSIDrive, devices map[string]*sys.Device) (bool, bool) {
	switch {
	case isHWInfoAvailable(drive):
		return syncDrive(ctx, devices, drive, matchDeviceHWInfo, "hardware IDs")
	case isDMMDUUIDAvailable(drive):
		return syncDrive(ctx, devices, drive, matchDeviceDMMDUUID, "DM/MD UUIDs")
	case isPTUUIDAvailable(drive):
		return syncDrive(ctx, devices, drive, matchDevicePTUUID, "Partition Table UUID")
	case isPartUUIDAvailable(drive):
		return syncDrive(ctx, devices, drive, matchDevicePartUUID, "Partition UUID")
	case isFSUUIDAvailable(drive):
		return syncDrive(ctx, devices, drive, matchDeviceFSUUID, "Fileystem UUIDs")
	case isV1Beta1Drive(drive):
		return syncDrive(ctx, devices, drive, matchV1Beta1Name, "v1beta1 drive name")
	default:
		return syncDrive(ctx, devices, drive, matchDeviceNameSize, "Device name and size")
	}
}

func removeDrive(ctx context.Context, drive *directcsi.DirectCSIDrive, devices map[string]*sys.Device) {
	for _, device := range devices {
		remove := func(matchFunc func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool) {
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
				if err := client.DeleteDrive(ctx, drive, true); err != nil {
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
			remove(func(drive *directcsi.DirectCSIDrive, device *sys.Device) bool {
				return drive.Status.Path == "/dev/"+device.Name && drive.Status.DriveStatus != directcsi.DriveStatusInUse
			})
		}
	}
}

func (handler *ueventHandler) processEvent(ctx context.Context, device *sys.Device, action string) {
	handler.syncMu.Lock()
	defer handler.syncMu.Unlock()

	resultCh, err := client.ListDrives(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(handler.nodeID)},
		[]utils.LabelValue{utils.NewLabelValue(device.Name)},
		nil,
		client.MaxThreadCount,
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

		drive := &result.Drive

		if action == uevent.Remove {
			removeDrive(ctx, drive, devices)
		} else {
			if matched, updated := updateDrive(ctx, drive, devices); matched {
				switch result.Drive.Status.DriveStatus {
				case directcsi.DriveStatusReady, directcsi.DriveStatusInUse:
					if updated {
						mountDrive(ctx, &result.Drive)
					}
				}
			}
		}

		if len(devices) == 0 {
			return
		}
	}

	if action == uevent.Remove {
		klog.Errorf("no drives found to remove by IDs for device %v", device.Name)
		return
	}

	drive := client.NewDirectCSIDrive(
		uuid.New().String(),
		client.NewDirectCSIDriveStatus(device, handler.nodeID, handler.topology),
	)
	if err := client.CreateDrive(ctx, drive); err != nil {
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
			return errors.New("canceled by context")
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

func (handler *ueventHandler) processLoop(ctx context.Context) error {
	syncFunc := func() {
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
	}

	if !handler.dynamicDriveDiscovery {
		syncFunc()
		return nil // This never happens.
	}

	go syncFunc()

	klog.V(3).Info("Starting uevent handler")

	for {
		event, err := handler.get(ctx)
		if err != nil {
			klog.Error(err)
			return err
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

func StartDynamicDriveHandler(ctx context.Context, identity, nodeID, rack, zone, region string, loopbackOnly bool) error {
	handler := &ueventHandler{
		nodeID: nodeID,
		topology: map[string]string{
			string(utils.TopologyDriverIdentity): identity,
			string(utils.TopologyDriverRack):     rack,
			string(utils.TopologyDriverZone):     zone,
			string(utils.TopologyDriverRegion):   region,
			string(utils.TopologyDriverNode):     nodeID,
		},
		dynamicDriveDiscovery: true,
		loopbackOnly:          loopbackOnly,
	}

	return handler.processLoop(ctx)
}
