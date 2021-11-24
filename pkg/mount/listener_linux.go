//go:build linux

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

package mount

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"k8s.io/klog/v2"
)

const (
	resyncTimeout        = int(10 * time.Minute / time.Millisecond)
	epollet       uint32 = syscall.EPOLLET & 0xffffffff
)

func toMountMap(mountInfo map[string][]Info) map[string]Info {
	mountMap := make(map[string]Info)
	for _, mountInfoList := range mountInfo {
		for _, mountInfo := range mountInfoList {
			mountMap[mountInfo.MountPoint] = mountInfo
		}
	}
	return mountMap
}

type Listener struct {
	epollFd        int
	mountFile      *os.File
	isClosed       int32
	prevMountInfo  map[string][]Info
	mountInfoMutex sync.Mutex
	closeCh        chan struct{}
	mutex          sync.Mutex
	eventMap       map[string]*Event
	keys           []string
	waitCh         chan struct{}
	infoCh         chan info
}

func (listener *Listener) Close() error {
	if atomic.AddInt32(&listener.isClosed, 1) == 1 {
		if listener.mountFile != nil {
			listener.mountFile.Close()
		}
		if listener.waitCh != nil {
			close(listener.waitCh)
		}
		close(listener.closeCh)
		close(listener.infoCh)
		return syscall.Close(listener.epollFd)
	}
	return nil
}

func (listener *Listener) get() (string, *Event, <-chan struct{}, error) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	if len(listener.eventMap) == 0 && len(listener.keys) > 0 {
		listener.keys = []string{}
	}

	var event *Event
	var found bool
	for !found {
		if len(listener.keys) == 0 {
			break
		}

		key := listener.keys[0]
		listener.keys = listener.keys[1:]
		if event, found = listener.eventMap[key]; found {
			delete(listener.eventMap, key)
			return key, event, nil, nil
		}
	}

	// As no event found, wait for event to be set.
	waitCh := listener.waitCh
	if waitCh == nil {
		waitCh = make(chan struct{})
		listener.waitCh = waitCh
	}
	return "", nil, waitCh, nil
}

func (listener *Listener) Get(ctx context.Context) (string, *Event, error) {
	for {
		key, event, waitCh, err := listener.get()
		switch {
		case err != nil:
			return "", nil, err
		case event != nil && key != "":
			return key, event, nil
		}

		select {
		case <-ctx.Done():
			return "", nil, fmt.Errorf("canceled by context; %w", ctx.Err())
		case <-listener.closeCh:
			return "", nil, errors.New("closed listener")
		case <-waitCh:
		}
	}
}

func (listener *Listener) set(mountPoint string, event *Event) {
	listener.mutex.Lock()
	defer listener.mutex.Unlock()

	listener.keys = append(listener.keys, mountPoint)
	listener.eventMap[mountPoint] = event

	if listener.waitCh != nil {
		close(listener.waitCh)
		listener.waitCh = nil
	}
}

func (listener *Listener) process(ctx context.Context) {
	isModified := func(currMountInfo, prevMountInfo Info) bool {
		switch {
		case currMountInfo.fsType != prevMountInfo.fsType:
			return true
		case currMountInfo.MajorMinor != prevMountInfo.MajorMinor:
			return true
		case !reflect.DeepEqual(currMountInfo.MountOptions, prevMountInfo.MountOptions):
			return true
		}
		return false
	}

	go func() {
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-listener.closeCh:
				return
			case info, ok := <-listener.infoCh:
				if !ok {
					return
				}
				currentMountMap := toMountMap(info.currentMountInfo)
				previousMountMap := toMountMap(info.previousMountInfo)
				for currMountPoint, currMountInfo := range currentMountMap {
					if prevMountInfo, ok := previousMountMap[currMountPoint]; !ok {
						listener.set(currMountPoint, &Event{
							eventType: Attached,
							mountInfo: currMountInfo,
						})
					} else {
						if isModified(currMountInfo, prevMountInfo) {
							listener.set(currMountPoint, &Event{
								eventType: Modified,
								mountInfo: currMountInfo,
							})
						}
						delete(previousMountMap, currMountPoint)
					}
				}
				for prevMountPoint, previousMountInfo := range previousMountMap {
					listener.set(prevMountPoint, &Event{
						eventType: Detached,
						mountInfo: previousMountInfo,
					})
				}
			}
		}
	}()
}

func (listener *Listener) setPreviousMountInfo(currentMountInfo map[string][]Info) {
	listener.mountInfoMutex.Lock()
	defer listener.mountInfoMutex.Unlock()
	listener.prevMountInfo = currentMountInfo
}

func (listener *Listener) start(ctx context.Context) {
	go func() {
		events := make([]syscall.EpollEvent, 16)
		defer listener.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-listener.closeCh:
				return
			default:
				if err := syscall.SetNonblock(int(listener.mountFile.Fd()), true); err != nil {
					klog.V(3).InfoS("unable to listen for mount events", "err", err)
					continue
				}
				_, err := syscall.EpollWait(listener.epollFd, events[:], resyncTimeout)
				if err != nil {
					klog.V(3).InfoS("unable to epoll_wait", "err", err)
					continue
				}
				currentMountInfo, err := Probe()
				if err != nil {
					klog.V(3).InfoS("unable to probe mounts while monitoring events", "err", err)
					continue
				}
				listener.infoCh <- info{
					previousMountInfo: listener.prevMountInfo,
					currentMountInfo:  currentMountInfo,
				}
				listener.setPreviousMountInfo(currentMountInfo)
			}
		}
	}()
}

func StartListener(ctx context.Context) (*Listener, error) {
	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(mountInfoProcFile)
	if err != nil {
		return nil, err
	}

	err = syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, int(f.Fd()), &syscall.EpollEvent{
		Events: syscall.EPOLLIN | epollet,
		Fd:     int32(f.Fd()),
	})
	if err != nil {
		return nil, err
	}

	mountInfo, err := Probe()
	if err != nil {
		return nil, err
	}

	eventMap := make(map[string]*Event)
	listener := &Listener{
		epollFd:       epfd,
		mountFile:     f,
		closeCh:       make(chan struct{}),
		eventMap:      eventMap,
		prevMountInfo: mountInfo,
		infoCh:        make(chan info, 16),
	}

	// start listening
	listener.process(ctx)
	listener.start(ctx)

	return listener, nil
}
