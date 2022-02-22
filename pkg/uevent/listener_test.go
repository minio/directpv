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

// func TestListenerGet(t *testing.T) {
// 	socketName, listener, serverConn, clientConn, sockfd, err := setupTestServer()
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	defer func() {
// 		clientConn.Close()
// 		serverConn.Close()
// 		listener.Close()
// 		os.Remove(socketName)
// 	}()

// 	eventListener := &Listener{
// 		closeCh:  make(chan struct{}),
// 		sockfd:   sockfd,
// 		eventMap: map[string]map[string]string{},
// 	}
// 	eventListener.start()
// 	defer eventListener.Close()

// 	case1Msg := append(getHeader(), []byte(strings.Join(
// 		[]string{
// 			".ID_FS_TYPE_NEW=",
// 			"ACTION=change",
// 			"DEVNAME=/dev/loop0",
// 			"DEVPATH=/devices/virtual/block/loop0",
// 			"DEVTYPE=disk",
// 			"ID_FS_TYPE=",
// 			"MAJOR=7",
// 			"MINOR=0",
// 			"SEQNUM=17050",
// 			"SUBSYSTEM=block",
// 			"TAGS=:systemd:",
// 			"USEC_INITIALIZED=132131168299",
// 		},
// 		string(fieldDelimiter),
// 	))...)
// 	case1Result := map[string]string{
// 		".ID_FS_TYPE_NEW":  "",
// 		"ACTION":           "change",
// 		"DEVNAME":          "/dev/loop0",
// 		"DEVPATH":          "/devices/virtual/block/loop0",
// 		"DEVTYPE":          "disk",
// 		"ID_FS_TYPE":       "",
// 		"MAJOR":            "7",
// 		"MINOR":            "0",
// 		"SEQNUM":           "17050",
// 		"SUBSYSTEM":        "block",
// 		"TAGS":             ":systemd:",
// 		"USEC_INITIALIZED": "132131168299",
// 	}

// 	if _, err := serverConn.Write(case1Msg); err != nil {
// 		t.Fatal(err)
// 	}

// 	result, err := eventListener.Get(context.TODO())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if !reflect.DeepEqual(result, case1Result) {
// 		t.Fatalf("result: expected: %v; got: %v", case1Result, result)
// 	}
// }
// fillMissingUdevData(runUdevData *sys.UDevData) error {

// Path         string
// Major        int
// Minor        int
// Partition    int
// WWID         string
// Model        string
// UeventSerial string
// Vendor       string
// DMName       string
// DMUUID       string
// MDUUID       string
// PTUUID       string
// PTType       string
// PartUUID     string
// UeventFSUUID string
// FSType       string
// FSUUID       string

// type deviceEvent struct {
// 	created time.Time
// 	action  action
// 	devPath string
// 	backOff time.Duration
// 	popped  bool
// 	timer   *time.Timer

// 	udevData *sys.UDevData
// }

func TestFillMissingUdevData(t *testing.T) {
	testCases := []struct {
		ueventUDevData   *sys.UDevData
		runUDevData      *sys.UDevData
		expectedUDevData *sys.UDevData
		expectedErr      error
	}{
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
			},
			runUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
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
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
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
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     203,
				Minor:     1,
				Partition: 1,
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "major", 202, 203),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     2,
				Partition: 1,
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "minor", 1, 2),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				WWID:      "WWID",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				WWID:      "ID",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				WWID:      "WWID",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "WWID", "WWID", "ID"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Model:     "modelnumber",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Model:     "invalidmodelnumber",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Model:     "modelnumber",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "Model", "modelnumber", "invalidmodelnumber"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventSerial: "ueventserial",
			},
			runUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventSerial: "invalidueventserial",
			},
			expectedUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventSerial: "ueventserial",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "UeventSerial", "ueventserial", "invalidueventserial"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Vendor:    "vendor",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Vendor:    "invalidvendor",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				Vendor:    "vendor",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "Vendor", "vendor", "invalidvendor"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMName:    "dmname",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMName:    "invaliddmname",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMName:    "dmname",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "DMName", "dmname", "invaliddmname"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMUUID:    "dmuuid",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMUUID:    "invaliddmuuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				DMUUID:    "dmuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "DMUUID", "dmuuid", "invaliddmuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				MDUUID:    "mduuid",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				MDUUID:    "invalidmduuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				MDUUID:    "mduuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "MDUUID", "mduuid", "invalidmduuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTUUID:    "ptuuid",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTUUID:    "invalidptuuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTUUID:    "ptuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PTUUID", "ptuuid", "invalidptuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTType:    "pttype",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTType:    "invalidpttype",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PTType:    "pttype",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PTType", "pttype", "invalidpttype"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PartUUID:  "partuuid",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PartUUID:  "invalidpartuuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				PartUUID:  "partuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "PartUUID", "partuuid", "invalidpartuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventFSUUID: "ueventfsuuid",
			},
			runUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventFSUUID: "invalidueventfsuuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:         "/dev/nvmen1p1",
				Major:        202,
				Minor:        1,
				Partition:    1,
				UeventFSUUID: "ueventfsuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "UeventFSUUID", "ueventfsuuid", "invalidueventfsuuid"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSType:    "fstype",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSType:    "invalidueventfstype",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSType:    "fstype",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "FSType", "fstype", "invalidueventfstype"),
		},
		{
			ueventUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSUUID:    "fsuuid",
			},
			runUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSUUID:    "invalidueventfsuuid",
			},
			expectedUDevData: &sys.UDevData{
				Path:      "/dev/nvmen1p1",
				Major:     202,
				Minor:     1,
				Partition: 1,
				FSUUID:    "fsuuid",
			},
			expectedErr: errValueMismatch("/dev/nvmen1p1", "FSUUID", "fsuuid", "invalidueventfsuuid"),
		},
	}

	for i, testCase := range testCases {
		dE := &deviceEvent{
			udevData: testCase.ueventUDevData,
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
