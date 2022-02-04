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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/minio/directpv/pkg/sys"
	"k8s.io/klog/v2"
)

type action string

const (
	libudev      = "libudev\x00"
	libudevMagic = 0xfeedcafe
	minMsgLen    = 40

	Add    action = "add"
	Change action = "change"
	Remove action = "remove"
)

var (
	errNonDeviceEvent = errors.New("event is not for a block device")
	errInvalidDevPath = errors.New("devpath not found in the event")
	pageSize          = os.Getpagesize()
	fieldDelimiter    = []byte{0}

	errEmptyBuf  = errors.New("buffer is empty")
	errShortRead = errors.New("short read")
)

type listener struct {
	sockfd      int
	eventQueue  *eventQueue
	threadiness int

	nodeID  string
	handler DeviceUEventHandler
}

type deviceEvent struct {
	created time.Time
	action  action
	devPath string
	backOff time.Duration
	popped  bool
	timer   *time.Timer

	udevData *sys.UDevData
}

func (dEvent *deviceEvent) fillMissingUdevData(runUdevData *sys.UDevData) error {
	errValueMismatch := func(path, key string, expected, found interface{}) error {
		return fmt.Errorf(
			"value mismatch for path %s. expected '%s': %v, received: %v",
			path,
			key,
			expected,
			found,
		)
	}

	// check for consistent fields
	if dEvent.udevData.Path != runUdevData.Path {
		return errValueMismatch(dEvent.udevData.Path, "path", dEvent.udevData.Path, runUdevData.Path)
	}
	if dEvent.udevData.Major != runUdevData.Major {
		return errValueMismatch(dEvent.udevData.Path, "major", dEvent.udevData.Major, runUdevData.Major)
	}
	if dEvent.udevData.Minor != runUdevData.Minor {
		return errValueMismatch(dEvent.udevData.Path, "minor", dEvent.udevData.Minor, runUdevData.Minor)
	}
	if dEvent.udevData.Partition != runUdevData.Partition {
		return errValueMismatch(dEvent.udevData.Path, "partitionnum", dEvent.udevData.Partition, runUdevData.Partition)
	}

	// Alternate pattern :-
	//
	// if runUdevData.WWID != "" {
	// 	switch dEvent.udevData.WWID {
	// 	case "":
	// 		dEvent.udevData.WWID = runUdevData.WWID
	// 	case runUdevData.WWID:
	// 	default:
	// 		errValueMismatch(dEvent.udevData.WWID, "WWID", dEvent.udevData.WWID, runUdevData.WWID)
	// 	}
	// }
	//

	if runUdevData.WWID != "" {
		if dEvent.udevData.WWID == "" {
			dEvent.udevData.WWID = runUdevData.WWID
		} else if dEvent.udevData.WWID != runUdevData.WWID {
			return errValueMismatch(dEvent.udevData.WWID, "WWID", dEvent.udevData.WWID, runUdevData.WWID)
		}
	}
	if runUdevData.Model != "" {
		if dEvent.udevData.Model == "" {
			dEvent.udevData.Model = runUdevData.Model
		} else if dEvent.udevData.Model != runUdevData.Model {
			return errValueMismatch(dEvent.udevData.Model, "Model", dEvent.udevData.Model, runUdevData.Model)
		}
	}
	if runUdevData.UeventSerial != "" {
		if dEvent.udevData.UeventSerial == "" {
			dEvent.udevData.UeventSerial = runUdevData.UeventSerial
		} else if dEvent.udevData.UeventSerial != runUdevData.UeventSerial {
			return errValueMismatch(dEvent.udevData.UeventSerial, "UeventSerial", dEvent.udevData.UeventSerial, runUdevData.UeventSerial)
		}
	}
	if runUdevData.Vendor != "" {
		if dEvent.udevData.Vendor == "" {
			dEvent.udevData.Vendor = runUdevData.Vendor
		} else if dEvent.udevData.Vendor != runUdevData.Vendor {
			return errValueMismatch(dEvent.udevData.Vendor, "Vendor", dEvent.udevData.Vendor, runUdevData.Vendor)
		}
	}
	if runUdevData.DMName != "" {
		if dEvent.udevData.DMName == "" {
			dEvent.udevData.DMName = runUdevData.DMName
		} else if dEvent.udevData.DMName != runUdevData.DMName {
			return errValueMismatch(dEvent.udevData.DMName, "DMName", dEvent.udevData.DMName, runUdevData.DMName)
		}
	}
	if runUdevData.DMUUID != "" {
		if dEvent.udevData.DMUUID == "" {
			dEvent.udevData.DMUUID = runUdevData.DMUUID
		} else if dEvent.udevData.DMUUID != runUdevData.DMUUID {
			return errValueMismatch(dEvent.udevData.DMUUID, "DMUUID", dEvent.udevData.DMUUID, runUdevData.DMUUID)
		}
	}
	if runUdevData.MDUUID != "" {
		if dEvent.udevData.MDUUID == "" {
			dEvent.udevData.MDUUID = runUdevData.MDUUID
		} else if dEvent.udevData.MDUUID != runUdevData.MDUUID {
			return errValueMismatch(dEvent.udevData.MDUUID, "MDUUID", dEvent.udevData.MDUUID, runUdevData.MDUUID)
		}
	}
	if runUdevData.PTUUID != "" {
		if dEvent.udevData.PTUUID == "" {
			dEvent.udevData.PTUUID = runUdevData.PTUUID
		} else if dEvent.udevData.PTUUID != runUdevData.PTUUID {
			return errValueMismatch(dEvent.udevData.PTUUID, "PTUUID", dEvent.udevData.PTUUID, runUdevData.PTUUID)
		}
	}
	if runUdevData.PTType != "" {
		if dEvent.udevData.PTType == "" {
			dEvent.udevData.PTType = runUdevData.PTType
		} else if dEvent.udevData.PTType != runUdevData.PTType {
			return errValueMismatch(dEvent.udevData.PTType, "PTType", dEvent.udevData.PTType, runUdevData.PTType)
		}
	}
	if runUdevData.PartUUID != "" {
		if dEvent.udevData.PartUUID == "" {
			dEvent.udevData.PartUUID = runUdevData.PartUUID
		} else if dEvent.udevData.PartUUID != runUdevData.PartUUID {
			return errValueMismatch(dEvent.udevData.PartUUID, "PartUUID", dEvent.udevData.PartUUID, runUdevData.PartUUID)
		}
	}
	if runUdevData.UeventFSUUID != "" {
		if dEvent.udevData.UeventFSUUID == "" {
			dEvent.udevData.UeventFSUUID = runUdevData.UeventFSUUID
		} else if dEvent.udevData.UeventFSUUID != runUdevData.UeventFSUUID {
			return errValueMismatch(dEvent.udevData.UeventFSUUID, "UeventFSUUID", dEvent.udevData.UeventFSUUID, runUdevData.UeventFSUUID)
		}
	}
	if runUdevData.FSType != "" {
		if dEvent.udevData.FSType == "" {
			dEvent.udevData.FSType = runUdevData.FSType
		} else if dEvent.udevData.FSType != runUdevData.FSType {
			return errValueMismatch(dEvent.udevData.FSType, "FSType", dEvent.udevData.FSType, runUdevData.FSType)
		}
	}
	if runUdevData.FSUUID != "" {
		if dEvent.udevData.FSUUID == "" {
			dEvent.udevData.FSUUID = runUdevData.FSUUID
		} else if dEvent.udevData.FSUUID != runUdevData.FSUUID {
			return errValueMismatch(dEvent.udevData.FSUUID, "FSUUID", dEvent.udevData.FSUUID, runUdevData.FSUUID)
		}
	}

	return nil
}

func (dEvent *deviceEvent) collectUDevData() error {
	switch dEvent.action {
	case Add, Change:
		// Older kernels like in CentOS 7 does not send all information about the device,
		// hence read relevant data from /run/udev/data/b<major>:<minor>
		runUdevDataMap, err := sys.ReadRunUdevDataFile(dEvent.udevData.Major, dEvent.udevData.Minor)
		if err != nil {
			return err
		}
		runUdevData, err := mapToUdevData(runUdevDataMap)
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

func (dEvent *deviceEvent) toDevice() (*sys.Device, error) {
	switch dEvent.action {
	case Add, Change:
		return sys.CreateDevice(dEvent.udevData)
	case Remove:
		// Removed device cannot be probed locally
		return sys.NewDevice(dEvent.udevData)
	default:
		return nil, fmt.Errorf("invalid device action: %s", dEvent.action)
	}
}

type DeviceUEventHandler interface {
	Add(context.Context, *sys.Device) error
	Change(context.Context, *sys.Device) error
	Remove(context.Context, *sys.Device) error
}

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
	}

	listener.processEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dEvent, err := listener.getNextDeviceUEvent(ctx)
			if err != nil {
				return err
			}
			listener.eventQueue.push(dEvent)
		}
	}
}

func (l *listener) handle(ctx context.Context, dEvent *deviceEvent) error {
	if sys.IsLoopBackDevice(dEvent.udevData.Path) {
		klog.V(5).InfoS(
			"loopback device is ignored",
			"ACTION", dEvent.action,
			"DEVPATH", dEvent.devPath)
		return nil
	}

	device, err := dEvent.toDevice()
	if err != nil {
		klog.ErrorS(err, "ACTION", dEvent.action, "DEVPATH", dEvent.devPath)
		return err
	}

	switch dEvent.action {
	case Add:
		return l.handler.Add(ctx, device)
	case Change:
		return l.handler.Change(ctx, device)
	case Remove:
		return l.handler.Remove(ctx, device)
	default:
		return fmt.Errorf("invalid device action: %s", dEvent.action)
	}
}

func (l *listener) processEvents(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				klog.Error(ctx.Err())
				return
			default:
				dEvent := l.eventQueue.pop()
				if err := dEvent.collectUDevData(); err != nil {
					klog.ErrorS(err, "failed to collect udevdata for path: %s", dEvent.devPath)
					l.eventQueue.push(dEvent)
					continue
				}
				if err := l.handle(ctx, dEvent); err != nil {
					klog.ErrorS(err, "failed to handle an event", dEvent.action)
					// Push it again to the queue
					l.eventQueue.push(dEvent)
				}
			}
		}
	}()
}

func (l *listener) getNextDeviceUEvent(ctx context.Context) (*deviceEvent, error) {
	for {
		buf, err := l.ReadMsg()
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

func parse(msg []byte) (map[string]string, error) {
	if !bytes.HasPrefix(msg, []byte(libudev)) {
		return nil, errors.New("libudev signature not found")
	}

	// magic number is stored in network byte order.
	if magic := binary.BigEndian.Uint32(msg[8:]); magic != libudevMagic {
		return nil, fmt.Errorf("libudev magic mismatch; expected: %v, got: %v", libudevMagic, magic)
	}

	offset := int(msg[16])
	if offset < 17 {
		return nil, fmt.Errorf("payload offset %v is not more than 17", offset)
	}
	if offset > len(msg) {
		return nil, fmt.Errorf("payload offset %v beyond message length %v", offset, len(msg))
	}

	fields := bytes.Split(msg[offset:], fieldDelimiter)
	event := map[string]string{}
	for _, field := range fields {
		if len(field) == 0 {
			continue
		}
		switch tokens := strings.SplitN(string(field), "=", 2); len(tokens) {
		case 1:
			event[tokens[0]] = ""
		case 2:
			event[tokens[0]] = tokens[1]
		}
	}
	return event, nil
}

func (l *listener) parseUEvent(buf []byte) (*deviceEvent, error) {
	eventMap, err := parse(buf)
	if err != nil {
		return nil, err
	}

	if eventMap["SUBSYSTEM"] != "block" {
		return nil, errNonDeviceEvent
	}

	eventAction := action(eventMap["ACTION"])
	switch eventAction {
	case Add, Change, Remove:
	default:
		return nil, fmt.Errorf("invalid device action: %s", eventAction)
	}

	udevData, err := mapToUdevData(eventMap)
	if err != nil {
		return nil, err
	}

	return &deviceEvent{
		created:  time.Now().UTC(),
		action:   eventAction,
		udevData: udevData,
		devPath:  udevData.Path,
	}, nil
}

func (l *listener) msgPeek() (int, []byte, error) {
	var n int
	var err error
	buf := make([]byte, os.Getpagesize())
	for {
		if n, _, err = syscall.Recvfrom(l.sockfd, buf, syscall.MSG_PEEK); err != nil {
			return n, nil, err
		}

		if n < len(buf) {
			break
		}

		buf = make([]byte, len(buf)+os.Getpagesize())
	}

	buf = buf[:n]

	return n, buf, err
}

func (l *listener) msgRead(buf []byte) error {
	if buf == nil {
		return errEmptyBuf
	}

	n, _, err := syscall.Recvfrom(l.sockfd, buf, 0)
	if err != nil {
		return err
	}

	if n != len(buf) {
		return errShortRead
	}

	return nil
}

// ReadMsg allow to read an entire uevent msg
func (l *listener) ReadMsg() ([]byte, error) {
	_, buf, err := l.msgPeek()
	if err != nil {
		return nil, err
	}
	if err = l.msgRead(buf); err != nil {
		return nil, err
	}

	return buf, nil
}
