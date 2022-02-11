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
	popped  bool
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
	return fmt.Sprintf("{created:%v, devPath:%v, action:%v}",
		d.created, d.devPath, d.action)
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
		keyCh:  make(chan string, 256),
	}
}

// pkg/node/devicehandler.go
// type handler interface{
//   Add()
//   Update()
//   Delete()
// }
//
// pkg/udev/uevent/listener.go
// ev := q.pop()
// if err := handle(ev); err != nil {
//   q.push(ev)
// }
//

func (e *eventQueue) set(event *deviceEvent) (existingEvent *deviceEvent, found bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	existingEvent, found = e.events[event.devPath]
	if found {
		if existingEvent.created.After(event.created) {
			// Skip as passed event is older.
			return nil, found
		}
	} else if event.popped {
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
				e.keyCh <- event.devPath
			},
		)
	}

	e.events[event.devPath] = event
	return existingEvent, found
}

func (e *eventQueue) push(event *deviceEvent) {
	existingEvent, found := e.set(event)

	if !found {
		if !event.popped {
			// As incoming event does not exist and it is new, send key to the channel
			e.keyCh <- event.devPath
		}
		return
	}

	if existingEvent == nil {
		// As existing event is newer than incoming event and not replaced by incoming event
		return
	}

	if !event.popped {
		if existingEvent.timer != nil {
			if existingEvent.timer.Stop() {
				// As incoming event is new and existing event has a timer and
				// we were able to stop the timer, send key to keyCh.
				e.keyCh <- event.devPath
			}
			// As we are not able to stop the timer i.e. the timer already elapsed,
			// the callback function in the timer sends key to keyCh.
		}
	}
}

func (e *eventQueue) pop() *deviceEvent {
	key := <-e.keyCh

	e.mutex.Lock()
	defer e.mutex.Unlock()

	if event, found := e.events[key]; found {
		delete(e.events, key)
		event.popped = true
		return event
	}

	panic("queue should also find event")
}
