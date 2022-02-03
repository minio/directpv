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

	Add    = "add"
	Change = "change"
	Remove = "remove"
)

var (
	pageSize          = os.Getpagesize()
	fieldDelimiter    = []byte{0}
	errClosedListener = errors.New("closed listener")
)

type key struct {
	name string
	err  error
}

type Listener struct {
	isClosed int32
	closeCh  chan struct{}
	sockfd   int
	mutex    sync.Mutex
	eventMap map[string]map[string]string
	keys     []key
	waitCh   chan struct{}

	handler UEventHandler
}

type UEventHandler interface {
	Handle(context.Context, map[string]string) error
}

func NewListener(handler UEventHandler) (*Listener, error) {
	sockfd, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return nil, err
	}

	err = syscall.Bind(sockfd, &syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
		Groups: 2,
	})
	if err != nil {
		return nil, err
	}

	listener := &Listener{
		closeCh:  make(chan struct{}),
		sockfd:   sockfd,
		eventMap: map[string]map[string]string{},
		handler:  handler,
	}
	return listener, nil
}

func (listener *Listener) Run(ctx context.Context) error {
	for {
		event, waitCh, err := listener.get(ctx)
		switch {
		case err != nil:
			return err
		case event != nil:
			if err := listener.handler.Handle(ctx, event); err != nil {
				// push it to end of queue
			}
		default:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-listener.closeCh:
				return errClosedListener
			case <-waitCh:
			}
		}
	}
}

func (listener *Listener) Close() error {
	if atomic.AddInt32(&listener.isClosed, 1) == 1 {
		close(listener.closeCh)
		return syscall.Close(listener.sockfd)
	}
	return nil
}

func (listener *Listener) get(ctx context.Context) (map[string]string, <-chan struct{}, error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	if len(listener.eventMap) == 0 && len(listener.keys) > 0 {
		listener.keys = []key{}
	}

	var event map[string]string
	var found bool
	for !found {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}

		if len(listener.keys) == 0 {
			break
		}

		key := listener.keys[0]
		listener.keys = listener.keys[1:]
		if key.err != nil {
			return nil, nil, key.err
		}
		if event, found = listener.eventMap[key.name]; found {
			delete(listener.eventMap, key.name)
			return event, nil, nil
		}
	}

	// As no event found, wait for event to be set.
	waitCh := listener.waitCh
	if waitCh == nil {
		waitCh = make(chan struct{})
		listener.waitCh = waitCh
	}
	return nil, waitCh, nil
}

func (listener *Listener) set(event map[string]string, err error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()
	if err != nil {
		listener.keys = append(listener.keys, key{err: err})
	} else {
		listener.keys = append(listener.keys, key{name: event["DEVNAME"]})
		listener.eventMap[event["DEVNAME"]] = event
	}

	if listener.waitCh != nil {
		close(listener.waitCh)
		listener.waitCh = nil
	}
}

func (listener *Listener) readData() ([]byte, error) {
	var err error
	var n int
	msg := make([]byte, pageSize)

	// Peek into the socket to know available bytes to read.
	for {
		select {
		case <-listener.closeCh:
			return nil, errClosedListener
		default:
			n, _, err = syscall.Recvfrom(listener.sockfd, msg, syscall.MSG_PEEK)
		}

		if err != nil {
			return nil, err
		}

		if n < len(msg) {
			msg = msg[:n]
			break
		}

		msg = make([]byte, len(msg)+pageSize)
	}

	// Read available bytes.
	select {
	case <-listener.closeCh:
		return nil, errClosedListener
	default:
		if n, _, err = syscall.Recvfrom(listener.sockfd, msg, 0); err != nil {
			return nil, err
		}
	}

	return msg[:n], nil
}

func (listener *Listener) read() (map[string]string, error) {
	for {
		msg, err := listener.readData()
		if err != nil {
			return nil, err
		}

		if len(msg) > minMsgLen {
			if event, err := parse(msg); err != nil {
				klog.Error(err)
			} else if event["SUBSYSTEM"] == "block" {
				return event, nil
			}
		}
	}
}

func (listener *Listener) Start() error {
	for {
		select {
		case <-listener.closeCh:
			return nil
		default:
			event, err := listener.read()
			if err != nil {
				klog.Error(err)
				return err
			}
			listener.set(event, err)
		}
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
