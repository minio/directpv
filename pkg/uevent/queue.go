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

func (e *eventQueue) push(event *deviceEvent) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	existingEvent, found := e.events[event.devPath]
	if found {
		// existing event created after incoming event
		if existingEvent.created.After(event.created) {
			// skip
			return
		}

		// existing event has backoff
		if existingEvent.timer != nil {
			// latest event preempts older event
			if !event.popped {
				// if we stopped the timer, and
				// prevented the push to keyCh
				if existingEvent.timer.Stop() {
					e.keyCh <- event.devPath
				}
			}
		}
		e.events[event.devPath] = event
		return
	}

	if event.popped {
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
	} else {
		e.keyCh <- event.devPath
	}
	e.events[event.devPath] = event
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
