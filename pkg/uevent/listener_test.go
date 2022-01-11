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
)

func TestParse(t *testing.T) {
	header := append([]byte(libudev), 0xfe, 0xed, 0xca, 0xfe, 0, 0, 0, 0)

	case3Msg := append(header, 16)
	case4Msg := append(header, 18)
	case5Msg := append(header, 17)
	case6Msg := append(header, 17, 'a')
	case7Msg := append(header, 17, 'a', '=', '1')
	case8Msg := append(header, 17, 0, 'a', 0, 'b', '=', '1', 0)

	testCases := []struct {
		msg            []byte
		expectedResult map[string]string
		expectErr      bool
	}{
		{nil, nil, true},
		{[]byte(libudev + "1234"), nil, true},
		{case3Msg, nil, true},
		{case4Msg, nil, true},
		{case5Msg, map[string]string{}, false},
		{case6Msg, map[string]string{"a": ""}, false},
		{case7Msg, map[string]string{"a": "1"}, false},
		{case8Msg, map[string]string{"a": "", "b": "1"}, false},
	}

	for i, testCase := range testCases {
		result, err := parse(testCase.msg)
		if testCase.expectErr {
			if err == nil {
				t.Fatalf("case %v: expected error; but succeeded", i+1)
			}
			continue
		}

		if !reflect.DeepEqual(result, testCase.expectedResult) {
			t.Fatalf("case %v: expected: %+v; got: %+v", i+1, testCase.expectedResult, result)
		}
	}
}

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

func TestListenerGet(t *testing.T) {
	socketName, listener, serverConn, clientConn, sockfd, err := setupTestServer()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		clientConn.Close()
		serverConn.Close()
		listener.Close()
		os.Remove(socketName)
	}()

	eventListener := &Listener{
		closeCh:  make(chan struct{}),
		sockfd:   sockfd,
		eventMap: map[string]map[string]string{},
	}
	eventListener.start()
	defer eventListener.Close()

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
	case1Result := map[string]string{
		".ID_FS_TYPE_NEW":  "",
		"ACTION":           "change",
		"DEVNAME":          "/dev/loop0",
		"DEVPATH":          "/devices/virtual/block/loop0",
		"DEVTYPE":          "disk",
		"ID_FS_TYPE":       "",
		"MAJOR":            "7",
		"MINOR":            "0",
		"SEQNUM":           "17050",
		"SUBSYSTEM":        "block",
		"TAGS":             ":systemd:",
		"USEC_INITIALIZED": "132131168299",
	}

	if _, err := serverConn.Write(case1Msg); err != nil {
		t.Fatal(err)
	}

	result, err := eventListener.Get(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(result, case1Result) {
		t.Fatalf("result: expected: %v; got: %v", case1Result, result)
	}
}
