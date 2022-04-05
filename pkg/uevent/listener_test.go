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
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/minio/directpv/pkg/sys"
)

func setupTestServer() (socketName string, listener net.Listener, serverConn, clientConn net.Conn, sockfd int, err error) {
	defer func() {
		if err != nil {
			if clientConn != nil {
				clientConn.Close()
			}

			if serverConn != nil {
				serverConn.Close()
			}

			if listener != nil {
				listener.Close()
			}

			os.Remove(socketName)
		}
	}()

	socketName = fmt.Sprintf("sock.%v", time.Now().UnixNano())
	if listener, err = net.Listen("unix", socketName); err != nil {
		return
	}

	doneCh := make(chan struct{})
	go func() {
		serverConn, err = listener.Accept()
		close(doneCh)
	}()

	var dialer net.Dialer
	dialer.LocalAddr = nil // if you have a local addr, add it here
	raddr := net.UnixAddr{Name: socketName, Net: "unix"}
	if clientConn, err = dialer.DialContext(context.TODO(), "unix", raddr.String()); err != nil {
		return
	}

	unixConn, ok := clientConn.(*net.UnixConn)
	if !ok {
		err = errors.New("clientConn is not (*net.UnixConn)")
		return
	}
	file, err := unixConn.File()
	if err != nil {
		return
	}
	sockfd = int(file.Fd())

	<-doneCh

	return socketName, listener, serverConn, clientConn, sockfd, nil
}

func getHeader() []byte {
	header := append([]byte(libudev), 0xfe, 0xed, 0xca, 0xfe, 0, 0, 0, 0, 40)
	pad := make([]byte, 23)
	return append(header, pad...)
}

func TestUEventListener(t *testing.T) {
	socketName, testListener, serverConn, clientConn, sockfd, err := setupTestServer()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()

	eventListener := &listener{
		sockfd:     sockfd,
		closeCh:    make(chan struct{}),
		eventQueue: newEventQueue(),
	}
	defer func() {
		eventListener.close(context.TODO())
		testListener.Close()
		clientConn.Close()
		serverConn.Close()
		os.Remove(socketName)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				t.Error("context canceled")
			default:
				dEvent, err := eventListener.getNextDeviceUEvent(ctx)
				if err == nil {
					eventListener.eventQueue.push(dEvent)
				}
			}
		}
	}()

	case1Msg := append(getHeader(), []byte(strings.Join(
		[]string{
			".ID_FS_TYPE_NEW=",
			"ACTION=change",
			"DEVNAME=/dev/loop0",
			"DEVPATH=/devices/virtual/block/loop0",
			"DEVTYPE=disk",
			"ID_FS_TYPE=",
			"MAJOR=7",
			"MINOR=0",
			"SEQNUM=17050",
			"SUBSYSTEM=block",
			"TAGS=:systemd:",
			"USEC_INITIALIZED=132131168299",
		},
		string(fieldDelimiter),
	))...)

	if _, err := serverConn.Write(case1Msg); err != nil {
		t.Fatal(err)
	}

	dEvent := eventListener.eventQueue.pop()

	if dEvent.devPath != "/dev/loop0" {
		t.Fatalf("expected devPath /dev/loop0 got %s", dEvent.devPath)
	}
	if dEvent.action != Change {
		t.Fatalf("expected action change got %s", string(dEvent.devPath))
	}
	if dEvent.major != 7 {
		t.Fatalf("expected major: 7 got %d", dEvent.major)
	}
	if dEvent.minor != 0 {
		t.Fatalf("expected major: 0 got %d", dEvent.minor)
	}

	case2Msg := append(getHeader(), []byte(strings.Join(
		[]string{
			".ID_FS_TYPE_NEW=",
			"ACTION=remove",
			"DEVNAME=/dev/loop0",
			"DEVPATH=/devices/virtual/block/loop0",
			"DEVTYPE=disk",
			"ID_FS_TYPE=",
			"MAJOR=7",
			"MINOR=0",
			"SEQNUM=17050",
			"SUBSYSTEM=block",
			"TAGS=:systemd:",
			"USEC_INITIALIZED=132131168299",
		},
		string(fieldDelimiter),
	))...)

	if _, err := serverConn.Write(case2Msg); err != nil {
		t.Fatal(err)
	}

	dEvent = eventListener.eventQueue.pop()

	if dEvent.devPath != "/dev/loop0" {
		t.Fatalf("expected devPath /dev/loop0 got %s", dEvent.devPath)
	}
	if dEvent.action != Remove {
		t.Fatalf("expected action change got %s", string(dEvent.devPath))
	}
	if dEvent.major != 7 {
		t.Fatalf("expected major: 7 got %d", dEvent.major)
	}
	if dEvent.minor != 0 {
		t.Fatalf("expected major: 0 got %d", dEvent.minor)
	}

	case3Msg := append(getHeader(), []byte(strings.Join(
		[]string{
			".ID_FS_TYPE_NEW=",
			"ACTION=add",
			"DEVNAME=/dev/loop0",
			"DEVPATH=/devices/virtual/block/loop0",
			"DEVTYPE=disk",
			"ID_FS_TYPE=",
			"MAJOR=7",
			"MINOR=0",
			"SEQNUM=17050",
			"SUBSYSTEM=block",
			"TAGS=:systemd:",
			"USEC_INITIALIZED=132131168299",
		},
		string(fieldDelimiter),
	))...)

	if _, err := serverConn.Write(case3Msg); err != nil {
		t.Fatal(err)
	}

	dEvent = eventListener.eventQueue.pop()

	if dEvent.devPath != "/dev/loop0" {
		t.Fatalf("expected devPath /dev/loop0 got %s", dEvent.devPath)
	}
	if dEvent.action != Add {
		t.Fatalf("expected action change got %s", string(dEvent.devPath))
	}
	if dEvent.major != 7 {
		t.Fatalf("expected major: 7 got %d", dEvent.major)
	}
	if dEvent.minor != 0 {
		t.Fatalf("expected major: 0 got %d", dEvent.minor)
	}
}

func TestFillMissingUdevData(t *testing.T) {
	testCases := []struct {
		ueventUDevData   *sys.UDevData
		runUDevData      *sys.UDevData
		expectedUDevData *sys.UDevData
		expectedErr      error
	}{
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
			},
			runUDevData: &sys.UDevData{
				Partition:    1,
				WWID:         "wwid",
				Model:        "model",
				UeventSerial: "serial",
				Vendor:       "vendor",
				DMName:       "dm-name",
				DMUUID:       "dm-uuid",
				PTUUID:       "ptuuid",
				PTType:       "pttype",
				PartUUID:     "part-uuid",
				UeventFSUUID: "fsuuid",
				FSType:       "xfs",
			},
			expectedUDevData: &sys.UDevData{
				Partition:    1,
				WWID:         "wwid",
				Model:        "model",
				UeventSerial: "serial",
				Vendor:       "vendor",
				DMName:       "dm-name",
				DMUUID:       "dm-uuid",
				PTUUID:       "ptuuid",
				PTType:       "pttype",
				PartUUID:     "part-uuid",
				UeventFSUUID: "fsuuid",
				FSType:       "xfs",
			},
			expectedErr: nil,
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				WWID:      "WWID",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				WWID:      "ID",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				WWID:      "WWID",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "WWID", "WWID", "ID"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				Model:     "modelnumber",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				Model:     "invalidmodelnumber",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				Model:     "modelnumber",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "Model", "modelnumber", "invalidmodelnumber"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition:    1,
				UeventSerial: "ueventserial",
			},
			runUDevData: &sys.UDevData{
				Partition:    1,
				UeventSerial: "invalidueventserial",
			},
			expectedUDevData: &sys.UDevData{
				Partition:    1,
				UeventSerial: "ueventserial",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "UeventSerial", "ueventserial", "invalidueventserial"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				Vendor:    "vendor",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				Vendor:    "invalidvendor",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				Vendor:    "vendor",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "Vendor", "vendor", "invalidvendor"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				DMName:    "dmname",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				DMName:    "invaliddmname",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				DMName:    "dmname",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "DMName", "dmname", "invaliddmname"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				DMUUID:    "dmuuid",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				DMUUID:    "invaliddmuuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				DMUUID:    "dmuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "DMUUID", "dmuuid", "invaliddmuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				MDUUID:    "mduuid",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				MDUUID:    "invalidmduuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				MDUUID:    "mduuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "MDUUID", "mduuid", "invalidmduuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				PTUUID:    "ptuuid",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				PTUUID:    "invalidptuuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				PTUUID:    "ptuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PTUUID", "ptuuid", "invalidptuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				PTType:    "pttype",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				PTType:    "invalidpttype",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				PTType:    "pttype",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PTType", "pttype", "invalidpttype"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				PartUUID:  "partuuid",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				PartUUID:  "invalidpartuuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				PartUUID:  "partuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PartUUID", "partuuid", "invalidpartuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition:    1,
				UeventFSUUID: "ueventfsuuid",
			},
			runUDevData: &sys.UDevData{
				Partition:    1,
				UeventFSUUID: "invalidueventfsuuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition:    1,
				UeventFSUUID: "ueventfsuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "UeventFSUUID", "ueventfsuuid", "invalidueventfsuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				FSType:    "fstype",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				FSType:    "invalidueventfstype",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				FSType:    "fstype",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "FSType", "fstype", "invalidueventfstype"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Partition: 1,
				FSUUID:    "fsuuid",
			},
			runUDevData: &sys.UDevData{
				Partition: 1,
				FSUUID:    "invalidueventfsuuid",
			},
			expectedUDevData: &sys.UDevData{
				Partition: 1,
				FSUUID:    "fsuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "FSUUID", "fsuuid", "invalidueventfsuuid"),
		},
	}

	for i, testCase := range testCases {
		dE := &deviceEvent{
			udevData: testCase.ueventUDevData,
			devPath:  "/dev/nvmen1p1",
		}
		err := dE.fillMissingUdevData(testCase.runUDevData)
		if err != nil {
			if testCase.expectedErr == nil {
				t.Fatalf("case %v: unexpected error: %v", i, err)
			} else {
				if err.Error() != testCase.expectedErr.Error() {
					t.Errorf("case %v: Expected err: %v but got %v", i, testCase.expectedErr, err)
				}
			}
		} else if testCase.expectedErr != nil {
			t.Errorf("case %v: Expected err: %v but got nil", i, testCase.expectedErr)
		}

		if !reflect.DeepEqual(dE.udevData, testCase.expectedUDevData) {
			t.Errorf("case %v: Expected udevdata: %v, got: %v", i, testCase.expectedUDevData, dE.udevData)
		}
	}
}
