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

package sys

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

const libudevMagic = uint32(0xfeedcafe)

// Uevent from Netlink
type Uevent struct {
	Header    string
	Action    string
	Devpath   string
	Subsystem string
	Seqnum    string
	Vars      map[string]string
}

type Decoder struct {
	c *Conn
}

func NewDecoder(c *Conn) *Decoder {
	return &Decoder{c}
}

// Decode - Parses the incoming uevent and extracts values
func (d *Decoder) ReadAndDecode() (*Uevent, error) {
	msg, err := d.c.ReadMsg()
	if err != nil {
		return nil, err
	}
	return ParseUEvent(msg)
}

func ParseUEvent(raw []byte) (e *Uevent, err error) {
	if len(raw) > 40 && bytes.Compare(raw[:8], []byte("libudev\x00")) == 0 {
		return parseUdevEvent(raw)
	}
	return
}

func parseUdevEvent(raw []byte) (e *Uevent, err error) {
	// the magic number is stored in network byte order.
	magic := binary.BigEndian.Uint32(raw[8:])
	if magic != libudevMagic {
		return nil, fmt.Errorf("cannot parse libudev event: magic number mismatch")
	}

	headerFields := bytes.Split(raw, []byte{0x00}) // 0x00 = end of string
	if len(headerFields) == 0 {
		err = fmt.Errorf("Wrong uevent format")
		return
	}

	e = &Uevent{
		Header: string(headerFields[0]),
	}

	// the payload offset int is stored in native byte order.
	payloadoff := *(*uint32)(unsafe.Pointer(&raw[16]))
	if payloadoff >= uint32(len(raw)) {
		return nil, fmt.Errorf("cannot parse libudev event: invalid data offset")
	}

	fields := bytes.Split(raw[payloadoff:], []byte{0x00}) // 0x00 = end of string
	if len(fields) == 0 {
		err = fmt.Errorf("cannot parse libudev event: data missing")
		return
	}

	envdata := make(map[string]string, 0)
	for _, envs := range fields[0 : len(fields)-1] {
		env := bytes.Split(envs, []byte("="))
		if len(env) != 2 {
			err = fmt.Errorf("cannot parse libudev event: invalid env data")
			return
		}
		envdata[string(env[0])] = string(env[1])
	}
	e.Vars = envdata

	for k, v := range envdata {
		switch k {
		case "ACTION":
			e.Action = v
		case "DEVPATH":
			e.Devpath = v
		case "SUBSYSTEM":
			e.Subsystem = v
		case "SEQNUM":
			e.Seqnum = v
		}
	}

	return
}

// Conn wraps the NETLINK fd.
type Conn struct {
	fd int
}

// ReadMsg allow to read an entire uevent msg
func (c *Conn) ReadMsg() (msg []byte, err error) {
	var n int

	buf := make([]byte, os.Getpagesize())
	for {
		// Just read how many bytes are available in the socket
		if n, _, err = syscall.Recvfrom(c.fd, buf, syscall.MSG_PEEK); err != nil {
			return
		}

		// If all message could be store inside the buffer : break
		if n < len(buf) {
			break
		}

		// Increase size of buffer if not enough
		buf = make([]byte, len(buf)+os.Getpagesize())
	}

	// Now read complete data
	n, _, err = syscall.Recvfrom(c.fd, buf, 0)
	if err != nil {
		return
	}

	// Extract only real data from buffer and return that
	msg = buf[:n]

	return
}

func (c *Conn) Close() error {
	return syscall.Close(c.fd)
}

// NewConn - Opens a NETLINK socket and binds it to the process.
func NewConn() (*Conn, error) {
	fd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_RAW,
		syscall.NETLINK_KOBJECT_UEVENT,
	)

	if err != nil {
		return nil, err
	}

	nl := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
		Groups: 2,
	}

	err = syscall.Bind(fd, &nl)
	return &Conn{fd}, err
}

// EventsCh - Returns a channel which emits the uevents from the fd.
func EventsCh(ctx context.Context, c *Conn, maxRetriesOnError int) <-chan Uevent {
	ueventCh := make(chan Uevent)
	var errC int

	dec := NewDecoder(c)
	go func() {
		defer close(ueventCh)
		for {
			if errC > maxRetriesOnError {
				log.Fatal("Retry limit exceeded. Stopping uevent listener")
				break
			}

			evt, err := dec.ReadAndDecode()
			if err != nil {
				log.Fatal(err)
				errC = errC + 1
				continue
			}

			errC = 0
			select {
			case ueventCh <- *evt:
				// Receive next event
			case <-ctx.Done():
				log.Fatal("Exiting on interrupt")
				return
			}
		}
	}()

	return ueventCh
}

// WatchBlockDevices - Watches for hotplugs and devices and returns a channel which emits block devices.
func WatchBlockDevices(ctx context.Context) (<-chan BlockDevice, error) {
	blockDeviceCh := make(chan BlockDevice)

	c, err := NewConn()
	if err != nil {
		return blockDeviceCh, err
	}

	eCh := EventsCh(ctx, c, 10)
	go func() {
		defer close(blockDeviceCh)
		for {
			select {
			case evt, ok := <-eCh:
				if !ok {
					// closed channel.
					return
				}
				// Filter subsystem
				if evt.Subsystem == "block" {
					// To-Do: Construct block device data
					blockDeviceCh <- BlockDevice{}
				}
			case <-ctx.Done():
				if err := c.Close(); err != nil {
					log.Printf("Error while closing reader %s", err.Error())
				}
				return
			}
		}
	}()

	return blockDeviceCh, err
}
