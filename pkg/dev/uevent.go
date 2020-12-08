// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package dev

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
)

// As per /usr/include/linux/netlink.h
const NETLINK_KOBJECT_UEVENT = 15

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
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{bufio.NewReader(r)}
}

// Decode - Parses the incoming uevent and extracts values
func (d *Decoder) Decode() (*Uevent, error) {
	ev := &Uevent{
		Vars: map[string]string{},
	}

	h, err := d.next()
	if err != nil {
		return nil, err
	}
	ev.Header = h

loop:
	for {
		kv, err := d.next()
		if err != nil {
			return nil, err
		}

		cleanKv := strings.TrimSpace(kv)
		parts := strings.Split(cleanKv, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("uevent format not supported: %s", kv)
		}
		k := parts[0]
		v := parts[1]

		ev.Vars[k] = v

		switch k {
		case "ACTION":
			ev.Action = v
		case "DEVPATH":
			ev.Devpath = v
		case "SUBSYSTEM":
			ev.Subsystem = v
		case "SEQNUM":
			ev.Seqnum = v
			break loop
		}
	}

	return ev, nil
}

// Read the next kv from the fd.
func (d *Decoder) next() (string, error) {
	s, err := d.r.ReadString(0x00)
	if err != nil {
		return "", err
	}
	return s, nil
}

// Reader wrapper for fd
type Reader struct {
	fd int
}

var _ io.ReadCloser = (*Reader)(nil)

func (r Reader) Read(p []byte) (n int, err error) {
	return syscall.Read(r.fd, p)
}

func (r Reader) Close() error {
	return syscall.Close(r.fd)
}

// NewReader - Opens a NETLINK socket and binds it to the process.
func NewReader() (io.ReadCloser, error) {
	fd, err := syscall.Socket(
		syscall.AF_NETLINK,
		syscall.SOCK_RAW,
		NETLINK_KOBJECT_UEVENT,
	)

	if err != nil {
		return nil, err
	}

	nl := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Pid:    uint32(os.Getpid()),
		Groups: 1,
	}

	err = syscall.Bind(fd, &nl)
	return &Reader{fd}, err
}

// EventCh - Returns a channel which emits the uevents from the fd.
func EventsCh(ctx context.Context, r io.Reader, maxRetriesOnError int) (error, <-chan Uevent) {
	ueventCh := make(chan Uevent)
	var errC int

	dec := NewDecoder(r)
	go func() {
		defer close(ueventCh)
		for {
			if errC > maxRetriesOnError {
				log.Fatal("Retry limit exceeded. Stopping uevent listener")
				break
			}

			evt, err := dec.Decode()
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

	return nil, ueventCh
}

// Watches for hotplugs and devices and returns a channel which emits block devices.
func WatchBlockDevices(ctx context.Context) (error, <-chan BlockDevice) {
	blockDeviceCh := make(chan BlockDevice)

	r, err := NewReader()
	if err != nil {
		return err, blockDeviceCh
	}

	err, eCh := EventsCh(ctx, r, 10)
	if err != nil {
		return err, blockDeviceCh
	}

	go func() {
		defer close(blockDeviceCh)
		for {
			select {
			case evt, ok := <-eCh:
				if !ok {
					// closed channel.
					return
				}
				fmt.Println("Received")
				fmt.Println("********************EVENT***********************************")
				fmt.Printf("\nAction: %s", evt.Action)
				fmt.Printf("\nDevpath: %s", evt.Devpath)
				fmt.Printf("\nSubsystem: %s", evt.Subsystem)
				fmt.Printf("\nSeqnum: %s", evt.Seqnum)
				fmt.Printf("\nVars: %v", evt.Vars)

				// Construct block device
				blockDeviceCh <- BlockDevice{}
			case <-ctx.Done():
				if err := r.Close(); err != nil {
					log.Printf("Error while closing reader %s", err.Error())
				}
				return
			}
		}
	}()

	return nil, blockDeviceCh
}
