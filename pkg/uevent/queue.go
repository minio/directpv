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
	"sync/atomic"
	"time"
)

const (
	defaultBackOff = 1 * time.Second
	maxBackOff     = 10 * time.Minute
)

type timer struct {
	duration     time.Duration
	stopCh       chan struct{}
	callbackFunc func()
	elapsedFlag  *int32
}

func (t timer) String() string {
	return fmt.Sprintf("timer{%v}", t.duration)
}

func (t *timer) stop() {
	close(t.stopCh)
}

func (t *timer) start() {
	go func() {
		select {
		case <-t.stopCh:
			return
		case <-time.After(t.duration):
			atomic.AddInt32(t.elapsedFlag, 1)
			t.callbackFunc()
		}
	}()
}

func (t *timer) elapsed() bool {
	return atomic.LoadInt32(t.elapsedFlag) != 0
}

func timeAfterFunc(duration time.Duration, callbackFunc func()) *timer {
	var i32 int32
	timer := &timer{
		duration:     duration,
		stopCh:       make(chan struct{}),
		callbackFunc: callbackFunc,
		elapsedFlag:  &i32,
	}
	timer.start()
	return timer
}

type deviceEvent struct {
	created time.Time
	devPath string
	action  string
	backOff time.Duration
	timer   *timer
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
	events       map[string]*deviceEvent
	keys         []string
	cond         *sync.Cond
	timers       []*timer
	stopTimersCh chan struct{}
}

func newEventQueue() *eventQueue {
	return &eventQueue{
		events: map[string]*deviceEvent{},
		cond:   sync.NewCond(&sync.Mutex{}),
	}
}

func (e *eventQueue) set(event *deviceEvent, addBackOff bool) {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()

	if existingEvent, found := e.events[event.devPath]; found && existingEvent.created.After(event.created) {
		// skip given event as existing event is the latest
		return
	}

	if event.timer != nil {
		event.timer.stop()
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
		event.timer = timeAfterFunc(event.backOff, func() { e.cond.Signal() })
	}

	e.events[event.devPath] = event
	found := false
	for _, key := range e.keys {
		if key == event.devPath {
			found = true
			break
		}
	}
	if !found {
		e.keys = append(e.keys, event.devPath)
	}

	e.cond.Signal()
}

func (e *eventQueue) push(event *deviceEvent) {
	e.set(event, false)
}

func (e *eventQueue) pushBackOff(event *deviceEvent) {
	e.set(event, true)
}

func (e *eventQueue) get() (event *deviceEvent) {
	e.cond.L.Lock()
	defer e.cond.L.Unlock()

	var keys []string
	for i, key := range e.keys {
		ev, found := e.events[key]
		if !found {
			continue
		}

		if ev.timer == nil || ev.timer.elapsed() {
			keys = append(keys, e.keys[i+1:]...)
			event = ev
			break
		}

		keys = append(keys, key)
	}
	e.keys = keys

	switch {
	case event != nil:
		delete(e.events, event.devPath)
	default:
		e.cond.Wait()
	}

	return event
}

func (e *eventQueue) pop() *deviceEvent {
	for {
		event := e.get()
		if event != nil {
			return event
		}
	}
}
