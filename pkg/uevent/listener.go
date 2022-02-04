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
	"sync"
	"sync/atomic"
	"syscall"

	"k8s.io/klog/v2"
)

const (
	libudev      = "libudev\x00"
	libudevMagic = 0xfeedcafe
	minMsgLen    = 40
)

var (
	pageSize       = os.Getpagesize()
	fieldDelimiter = []byte{0}

	errNonDeviceEvent = errors.New("Uevent is not for a block device")
	errEmptyBuf       = errors.New("buffer is empty")
)

type deviceEvent struct {
	path  string
	major int
	minor int
}

type listener struct {
	sockfd      int
	queue       *queue
	threadiness int

	handler DeviceUEventHandler
}

type DeviceUEventHandler interface {
	Change(context.Context, *sys.Device) error
	Delete(context.Context, *sys.Device) error
}

func Run(ctx context.Context, handler DeviceUEventHandler) error {
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
		sockfd:  sockfd,
		handler: handler,
		queue:   newQueue,
	}

	go listener.processEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event, err := listener.getNextDeviceUEvent(ctx)
			if err != nil {
				return err
			}
			listener.queue.Push(event)
		}
	}
}

func (l *Listener) getNextDeviceUEvent(ctx context.Context) (*deviceEvent, error) {
	for {
		buf, err := l.ReadMsg()
		if err != nil {
			return nil, err
		}

		dEv, err := l.unmarshalDeviceUevent(buf)
		if err != nil {
			if errors.Is(errNonBlockDevice) {
				continue
			}
			return nil, err
		}
		return dEv, nil
	}
}

func (l *listener) unmarshalDeviceUevent(buf []byte) (*deviceEvent, error) {
	return nil, nil
}

func (l *listener) msgPeek() (int, *[]byte, error) {
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
	return n, &buf, err
}

func (l *listener) msgRead(buf *[]byte) error {
	if buf == nil {
		return errEmptyBuf
	}

	n, _, err := syscall.Recvfrom(l.sockfd, *buf, 0)
	if err != nil {
		return err
	}

	*buf = (*buf)[:n]

	return nil
}

// ReadMsg allow to read an entire uevent msg
func (l *listener) ReadMsg() ([]byte, error) {
	_, buf, err := c.msgPeek()
	if err != nil {
		return nil, err
	}
	if err = c.msgRead(buf); err != nil {
		return nil, err
	}

	return *buf, nil
}
