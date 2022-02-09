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
	"fmt"
	"sync"
	"time"
)

const (
	defaultBackOff = 1 * time.Second
	maxBackOff     = 10 * time.Minute
)

type deviceEvent struct {
	created time.Time
	devPath string
	action  string
	backOff time.Duration
	timer   *time.Timer
}

func newDeviceEvent(devPath, action string) *deviceEvent {
	return &deviceEvent{
		created: time.Now().UTC(),
		devPath: devPath,
		action:  action,
	}
}

func (d deviceEvent) String() string {
	return fmt.Sprintf("{created:%v, devPath:%v, action:%v, timer:%v}", d.created, d.devPath, d.action, d.timer)
}

type eventQueue struct {
	events map[string]*deviceEvent
	mutex  sync.Mutex
	keyCh  chan string
	keys   sync.Map
}

func newEventQueue() *eventQueue {
	return &eventQueue{
		events: map[string]*deviceEvent{},
		keyCh:  make(chan string, 16384),
	}
}

func (e *eventQueue) set(event *deviceEvent, addBackOff bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if existingEvent, found := e.events[event.devPath]; found && existingEvent.created.After(event.created) {
		// skip given event as existing event is the latest
		return
	}

	if event.timer != nil {
		event.timer.Stop()
	}

	if addBackOff {
		switch event.backOff {
		case 0:
			event.backOff = defaultBackOff
		default:
			event.backOff *= 2
			if event.backOff > maxBackOff {
				event.backOff = maxBackOff
			}
		}
		event.timer = time.AfterFunc(
			event.backOff,
			func() {
				if _, loaded := e.keys.LoadOrStore(event.devPath, struct{}{}); !loaded {
					e.keyCh <- event.devPath
				}
			},
		)
	}

	e.events[event.devPath] = event

	if !addBackOff {
		if _, loaded := e.keys.LoadOrStore(event.devPath, struct{}{}); !loaded {
			e.keyCh <- event.devPath
		}
	}
}

func (e *eventQueue) push(event *deviceEvent) {
	e.set(event, false)
}

func (e *eventQueue) pushBackOff(event *deviceEvent) {
	e.set(event, true)
}

func (e *eventQueue) get(key string) *deviceEvent {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if event, found := e.events[key]; found {
		delete(e.events, key)
		return event
	}

	return nil
}

func (e *eventQueue) pop() *deviceEvent {
	for key := range e.keyCh {
		e.keys.LoadAndDelete(key)
		event := e.get(key)
		if event != nil {
			return event
		}
	}

	// This happens only if e.keyCh is closed.
	return nil
}
