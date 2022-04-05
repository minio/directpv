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

package uevent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type action string

const (
	libudev      = "libudev\x00"
	libudevMagic = 0xfeedcafe

	// Add event
	Add action = "add"
	// Change event
	Change action = "change"
	// Remove event
	Remove action = "remove"
	// Sync internal
	Sync action = "sync"
)

var (
	fieldDelimiter = []byte{0}
	resyncPeriod   = 30 * time.Second
	syncInterval   = 30 * time.Second
	// errors
	errEmptyBuf            = errors.New("buffer is empty")
	errShortRead           = errors.New("short read")
	errNonDeviceEvent      = errors.New("event is not for a block device")
	errTooManyMatchesFound = func(device *sys.Device, action action) error {
		return fmt.Errorf("too many matches found for device %s while processing %s", device.DevPath(), action)
	}
	errValueMismatch = func(path, key string, expected, found interface{}) error {
		return fmt.Errorf(
			"value mismatch for path %s. expected '%s': %v, received: %v",
			path,
			key,
			expected,
			found,
		)
	}
	errClosedListener = errors.New("closed listener")
)

// DeviceUEventHandler is an interface with uevent methods
type DeviceUEventHandler interface {
	Add(context.Context, *sys.Device) error
	Update(context.Context, *sys.Device, *directcsi.DirectCSIDrive) error
	Remove(context.Context, *directcsi.DirectCSIDrive) error
}

type listener struct {
	isClosed   int32
	closeCh    chan struct{}
	sockfd     int
	eventQueue *eventQueue

	nodeID  string
	handler DeviceUEventHandler

	indexer *indexer
}

type deviceEvent struct {
	created time.Time
	action  action
	major   int
	minor   int
	devPath string
	backOff time.Duration
	popped  bool
	timer   *time.Timer

	udevData *sys.UDevData
}

// Run listens for events
func Run(ctx context.Context, nodeID string, handler DeviceUEventHandler) error {
	sockfd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_RAW,
		syscall.NETLINK_KOBJECT_UEVENT,
	)
	if err != nil {
		return err
	}

	if err := syscall.Bind(sockfd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
		Groups: 2,
	}); err != nil {
		return err
	}

	listener := &listener{
		sockfd:     sockfd,
		handler:    handler,
		eventQueue: newEventQueue(),
		nodeID:     nodeID,
		indexer:    newIndexer(ctx, nodeID, resyncPeriod),
	}
	defer listener.close(ctx)

	go listener.startSync(ctx)

	go listener.processEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-listener.closeCh:
			return errClosedListener
		default:
			dEvent, err := listener.getNextDeviceUEvent(ctx)
			if err != nil {
				return err
			}
			listener.eventQueue.push(dEvent)
		}
	}
}

func (l *listener) close(ctx context.Context) error {
	if atomic.AddInt32(&l.isClosed, 1) == 1 {
		close(l.closeCh)
		return syscall.Close(l.sockfd)
	}
	return nil
}

func (l *listener) startSync(ctx context.Context) {
	if err := l.sync(); err != nil {
		klog.Errorf("error while sycing: %v", err)
	}
	syncTicker := time.NewTicker(syncInterval)
	defer syncTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			klog.Error(ctx.Err())
			return
		case <-l.closeCh:
			klog.Error(errClosedListener)
			return
		case <-syncTicker.C:
			if err := l.sync(); err != nil {
				klog.Errorf("error while sycing: %v", err)
			}
		}
	}
}

func (l *listener) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Error(ctx.Err())
			return
		case <-l.closeCh:
			klog.Error(errClosedListener)
			return
		default:
			dEvent := l.eventQueue.pop()
			if err := dEvent.collectUDevData(); err != nil {
				klog.ErrorS(err, "failed to collect udevdata for path: %s", dEvent.devPath)
				if dEvent.action != Sync {
					l.eventQueue.push(dEvent)
				}
				continue
			}
			if err := l.handle(ctx, dEvent); err != nil {
				klog.ErrorS(err, "failed to handle an event", dEvent.action)
				if dEvent.action != Sync {
					// Push it again to the queue
					l.eventQueue.push(dEvent)
				}
			}
		}
	}
}

func (l *listener) handle(ctx context.Context, dEvent *deviceEvent) error {
	if sys.IsLoopBackDevice(dEvent.devPath) {
		klog.V(5).InfoS(
			"loopback device is ignored",
			"ACTION", dEvent.action,
			"DEVPATH", dEvent.devPath)
		return nil
	}

	if dEvent.devPath == "" {
		return fmt.Errorf("udevData does not have valid DEVPATH %v", dEvent.devPath)
	}

	device := &sys.Device{
		Name:         filepath.Base(dEvent.devPath),
		Major:        dEvent.major,
		Minor:        dEvent.minor,
		Virtual:      strings.Contains(dEvent.devPath, "/virtual/"),
		Partition:    dEvent.udevData.Partition,
		WWID:         dEvent.udevData.WWID,
		Model:        dEvent.udevData.Model,
		UeventSerial: dEvent.udevData.UeventSerial,
		Vendor:       dEvent.udevData.Vendor,
		DMName:       dEvent.udevData.DMName,
		DMUUID:       dEvent.udevData.DMUUID,
		MDUUID:       dEvent.udevData.MDUUID,
		PTUUID:       dEvent.udevData.PTUUID,
		PTType:       dEvent.udevData.PTType,
		PartUUID:     dEvent.udevData.PartUUID,
		UeventFSUUID: dEvent.udevData.UeventFSUUID,
		FSType:       dEvent.udevData.FSType,
		PCIPath:      dEvent.udevData.PCIPath,
		SerialLong:   dEvent.udevData.UeventSerialLong,
	}

	if dEvent.action != Remove {
		if err := device.ProbeSysInfo(); err != nil {
			return err
		}
		if err := device.ProbeMountInfo(); err != nil {
			return err
		}
		ok, err := l.validateDevice(device)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := device.ProbeDevInfo(); err != nil {
			return err
		}
	}

	drives, err := l.indexer.listDrives()
	if err != nil {
		return err
	}
	drive, matchResult := runMatchers(drives, device, stageOneMatchers, stageTwoMatchers)

	switch dEvent.action {
	case Add:
		return l.processAdd(ctx, matchResult, device, drive)
	case Change, Sync:
		return l.processUpdate(ctx, matchResult, device, drive)
	case Remove:
		return l.processRemove(ctx, matchResult, device, drive)
	default:
		return fmt.Errorf("invalid device action: %s", dEvent.action)
	}
}

func (l *listener) validateDevice(device *sys.Device) (bool, error) {
	if device.UeventFSUUID == "" {
		// do the full probe for available and unavailable drives
		return false, nil
	}
	filteredDrives, err := l.indexer.filterDrivesByUEventFSUUID(device.UeventFSUUID)
	if err != nil {
		return false, err
	}
	if len(filteredDrives) != 1 {
		return false, nil
	}
	filteredDrive := filteredDrives[0]

	return !isFormatRequested(filteredDrive) &&
		ValidateMountInfo(device, filteredDrive) &&
		ValidateUDevInfo(device, filteredDrive) &&
		validateSysInfo(device, filteredDrive), nil
}

func (l *listener) processAdd(ctx context.Context,
	matchResult matchResult,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {
	switch matchResult {
	case noMatch:
		return l.handler.Add(ctx, device)
	case changed, noChange:
		klog.V(3).Infof("ignoring ADD action for the device %s as the corresponding drive match %s is found", device.DevPath(), drive.Name)
		return nil
	case tooManyMatches:
		return errTooManyMatchesFound(device, Add)
	default:
		return fmt.Errorf("invalid match result: %v", matchResult)
	}
}

func (l *listener) processUpdate(ctx context.Context,
	matchResult matchResult,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {
	switch matchResult {
	case noMatch:
		return l.handler.Add(ctx, device)
	case changed:
		return l.handler.Update(ctx, device, drive)
	case noChange:
		// check if the lost drive is back
		for i := range drive.Status.Conditions {
			if drive.Status.Conditions[i].Type == string(directcsi.DirectCSIDriveConditionReady) &&
				drive.Status.Conditions[i].Status == metav1.ConditionFalse &&
				drive.Status.Conditions[i].Reason == string(directcsi.DirectCSIDriveReasonLost) {
				utils.UpdateCondition(drive.Status.Conditions,
					string(directcsi.DirectCSIDriveConditionReady),
					metav1.ConditionTrue,
					string(directcsi.DirectCSIDriveReasonLost),
					"")
				_, err := client.GetLatestDirectCSIDriveInterface().Update(
					ctx, drive, metav1.UpdateOptions{TypeMeta: utils.DirectCSIDriveTypeMeta()},
				)
				if err != nil {
					return err
				}
			}
		}
		return nil
	case tooManyMatches:
		return errTooManyMatchesFound(device, Change)
	default:
		return fmt.Errorf("invalid match result: %v", matchResult)
	}
}

func (l *listener) processRemove(ctx context.Context,
	matchResult matchResult,
	device *sys.Device,
	drive *directcsi.DirectCSIDrive) error {
	switch matchResult {
	case noMatch:
		klog.V(3).InfoS(
			"matching drive not found",
			"ACTION", Remove,
			"DEVICE", device.Name)
		return nil
	case changed, noChange:
		return l.handler.Remove(ctx, drive)
	case tooManyMatches:
		return errTooManyMatchesFound(device, Remove)
	default:
		return fmt.Errorf("invalid match result: %v", matchResult)
	}
}

func (l *listener) getNextDeviceUEvent(ctx context.Context) (*deviceEvent, error) {
	for {
		select {
		case <-ctx.Done():
			klog.Error(ctx.Err())
			return nil, ctx.Err()
		case <-l.closeCh:
			return nil, errClosedListener
		default:
			buf, err := l.readMsg(ctx)
			if err != nil {
				return nil, err
			}
			dEvent, err := l.parseUEvent(buf)
			if err != nil {
				if errors.Is(err, errNonDeviceEvent) {
					continue
				}
				return nil, err
			}
			return dEvent, nil
		}
	}
}

func (dEvent *deviceEvent) collectUDevData() error {
	switch dEvent.action {
	case Add, Change, Sync:
		devName, err := sys.GetDeviceName(uint32(dEvent.major), uint32(dEvent.minor))
		if err != nil {
			return err
		}
		if filepath.Base(dEvent.devPath) != devName {
			return fmt.Errorf("path mismatch. Expected %s got %s", filepath.Base(dEvent.devPath), devName)
		}
		// Older kernels like in CentOS 7 does not send all information about the device,
		// hence read relevant data from /run/udev/data/b<major>:<minor>
		runUdevDataMap, err := sys.ReadRunUdevDataByMajorMinor(dEvent.major, dEvent.minor)
		if err != nil {
			return err
		}
		runUdevData, err := sys.MapToUdevData(runUdevDataMap)
		if err != nil {
			return err
		}
		// Fill the missing fields
		return dEvent.fillMissingUdevData(runUdevData)
	case Remove:
		// Removed device cannot be probed locally
		// Relying on the event data
		return nil
	default:
		return fmt.Errorf("invalid device action: %s", dEvent.action)
	}
}

func (dEvent *deviceEvent) fillMissingUdevData(runUdevData *sys.UDevData) error {
	// check for consistent fields
	if runUdevData.Partition != dEvent.udevData.Partition {
		if dEvent.udevData.Partition == 0 {
			dEvent.udevData.Partition = runUdevData.Partition
		} else {
			return errValueMismatch(dEvent.devPath, "partitionnum", dEvent.udevData.Partition, runUdevData.Partition)
		}
	}

	if runUdevData.WWID != "" {
		if dEvent.udevData.WWID == "" {
			dEvent.udevData.WWID = runUdevData.WWID
		} else if dEvent.udevData.WWID != runUdevData.WWID {
			return errValueMismatch(dEvent.devPath, "WWID", dEvent.udevData.WWID, runUdevData.WWID)
		}
	}
	if runUdevData.Model != "" {
		if dEvent.udevData.Model == "" {
			dEvent.udevData.Model = runUdevData.Model
		} else if dEvent.udevData.Model != runUdevData.Model {
			return errValueMismatch(dEvent.devPath, "Model", dEvent.udevData.Model, runUdevData.Model)
		}
	}
	if runUdevData.UeventSerial != "" {
		if dEvent.udevData.UeventSerial == "" {
			dEvent.udevData.UeventSerial = runUdevData.UeventSerial
		} else if dEvent.udevData.UeventSerial != runUdevData.UeventSerial {
			return errValueMismatch(dEvent.devPath, "UeventSerial", dEvent.udevData.UeventSerial, runUdevData.UeventSerial)
		}
	}
	if runUdevData.UeventSerialLong != "" {
		if dEvent.udevData.UeventSerialLong == "" {
			dEvent.udevData.UeventSerialLong = runUdevData.UeventSerialLong
		} else if dEvent.udevData.UeventSerialLong != runUdevData.UeventSerialLong {
			return errValueMismatch(dEvent.devPath, "UeventSerialLong", dEvent.udevData.UeventSerialLong, runUdevData.UeventSerialLong)
		}
	}
	if runUdevData.Vendor != "" {
		if dEvent.udevData.Vendor == "" {
			dEvent.udevData.Vendor = runUdevData.Vendor
		} else if dEvent.udevData.Vendor != runUdevData.Vendor {
			return errValueMismatch(dEvent.devPath, "Vendor", dEvent.udevData.Vendor, runUdevData.Vendor)
		}
	}
	if runUdevData.DMName != "" {
		if dEvent.udevData.DMName == "" {
			dEvent.udevData.DMName = runUdevData.DMName
		} else if dEvent.udevData.DMName != runUdevData.DMName {
			return errValueMismatch(dEvent.devPath, "DMName", dEvent.udevData.DMName, runUdevData.DMName)
		}
	}
	if runUdevData.DMUUID != "" {
		if dEvent.udevData.DMUUID == "" {
			dEvent.udevData.DMUUID = runUdevData.DMUUID
		} else if dEvent.udevData.DMUUID != runUdevData.DMUUID {
			return errValueMismatch(dEvent.devPath, "DMUUID", dEvent.udevData.DMUUID, runUdevData.DMUUID)
		}
	}
	if runUdevData.MDUUID != "" {
		if dEvent.udevData.MDUUID == "" {
			dEvent.udevData.MDUUID = runUdevData.MDUUID
		} else if dEvent.udevData.MDUUID != runUdevData.MDUUID {
			return errValueMismatch(dEvent.devPath, "MDUUID", dEvent.udevData.MDUUID, runUdevData.MDUUID)
		}
	}
	if runUdevData.PTUUID != "" {
		if dEvent.udevData.PTUUID == "" {
			dEvent.udevData.PTUUID = runUdevData.PTUUID
		} else if dEvent.udevData.PTUUID != runUdevData.PTUUID {
			return errValueMismatch(dEvent.devPath, "PTUUID", dEvent.udevData.PTUUID, runUdevData.PTUUID)
		}
	}
	if runUdevData.PTType != "" {
		if dEvent.udevData.PTType == "" {
			dEvent.udevData.PTType = runUdevData.PTType
		} else if dEvent.udevData.PTType != runUdevData.PTType {
			return errValueMismatch(dEvent.devPath, "PTType", dEvent.udevData.PTType, runUdevData.PTType)
		}
	}
	if runUdevData.PartUUID != "" {
		if dEvent.udevData.PartUUID == "" {
			dEvent.udevData.PartUUID = runUdevData.PartUUID
		} else if dEvent.udevData.PartUUID != runUdevData.PartUUID {
			return errValueMismatch(dEvent.devPath, "PartUUID", dEvent.udevData.PartUUID, runUdevData.PartUUID)
		}
	}
	if runUdevData.UeventFSUUID != "" {
		if dEvent.udevData.UeventFSUUID == "" {
			dEvent.udevData.UeventFSUUID = runUdevData.UeventFSUUID
		} else if dEvent.udevData.UeventFSUUID != runUdevData.UeventFSUUID {
			return errValueMismatch(dEvent.devPath, "UeventFSUUID", dEvent.udevData.UeventFSUUID, runUdevData.UeventFSUUID)
		}
	}
	if runUdevData.FSType != "" {
		if dEvent.udevData.FSType == "" {
			dEvent.udevData.FSType = runUdevData.FSType
		} else if dEvent.udevData.FSType != runUdevData.FSType {
			return errValueMismatch(dEvent.devPath, "FSType", dEvent.udevData.FSType, runUdevData.FSType)
		}
	}
	if runUdevData.FSUUID != "" {
		if dEvent.udevData.FSUUID == "" {
			dEvent.udevData.FSUUID = runUdevData.FSUUID
		} else if dEvent.udevData.FSUUID != runUdevData.FSUUID {
			return errValueMismatch(dEvent.devPath, "FSUUID", dEvent.udevData.FSUUID, runUdevData.FSUUID)
		}
	}
	if runUdevData.PCIPath != "" {
		if dEvent.udevData.PCIPath == "" {
			dEvent.udevData.PCIPath = runUdevData.PCIPath
		} else if dEvent.udevData.PCIPath != runUdevData.PCIPath {
			return errValueMismatch(dEvent.devPath, "PCIPath", dEvent.udevData.PCIPath, runUdevData.PCIPath)
		}
	}

	return nil
}

func (l *listener) sync() error {
	dir, err := os.Open("/run/udev/data")
	if err != nil {
		return err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		if !strings.HasPrefix(name, "b") {
			continue
		}

		major, minor, err := utils.GetMajorMinorFromStr(strings.TrimPrefix(name, "b"))
		if err != nil {
			klog.V(5).Infof("error while parsing maj:min for file: %s: %v", name, err)
			continue
		}
		devName, err := sys.GetDeviceName(major, minor)
		if err != nil {
			klog.V(5).Infof("error while getting device name for maj:min (%v:%v): %v", major, minor, err)
			continue
		}

		data, err := sys.ReadRunUdevDataByMajorMinor(int(major), int(minor))
		if err != nil {
			klog.V(5).Infof("error while reading udevdata for device %s: %v", devName, err)
			continue
		}

		runUdevData, err := sys.MapToUdevData(data)
		if err != nil {
			klog.V(5).Infof("error while mapping udevdata for device %s: %v", devName, err)
			continue
		}

		event := &deviceEvent{
			created:  time.Now().UTC(),
			action:   Sync,
			udevData: runUdevData,
			devPath:  "/dev/" + devName,
			major:    int(major),
			minor:    int(minor),
		}

		l.eventQueue.push(event)
	}

	return nil
}
