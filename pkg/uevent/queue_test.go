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
	"reflect"
	"sync"
	"testing"
)

func TestEventQueue(t *testing.T) {
	queue := newEventQueue()

	var expectedResult1, expectedResult2, result *deviceEvent

	// single event push and pop
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// multiple same device events push, but pop returns the latest event
	queue.push(newDeviceEvent("sda", "change"))
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// multiple different device events push, but pops return the latest events
	queue.push(newDeviceEvent("sda", "remove"))
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult2)
	queue.push(newDeviceEvent("sda", "change"))
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult2, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult2, result)
	}

	// with/without backoff pushes and second pop returns backoff pushed event
	expectedResult2 = newDeviceEvent("sda", "change")
	queue.pushBackOff(expectedResult2)
	expectedResult1 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult2, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult2, result)
	}

	// the latest event replaces backoff push
	queue.pushBackOff(newDeviceEvent("sda", "change"))
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// the latest event with backoff replaces older event
	queue.push(newDeviceEvent("sda", "change"))
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.pushBackOff(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// the latest event is not replaced by older event
	oldEvent := newDeviceEvent("sda", "change")
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.pushBackOff(expectedResult1)
	queue.pushBackOff(oldEvent)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
}

func TestEventQueueBackOff(t *testing.T) {
	queue := newEventQueue()

	var expectedResult1, expectedResult2, result *deviceEvent

	// all events with backoff
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.pushBackOff(expectedResult1)
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.pushBackOff(expectedResult2)
	var results []*deviceEvent
	results = append(results, queue.pop())
	results = append(results, queue.pop())
	if len(results) != 2 {
		t.Fatalf("expected: 2, got: %v of %v", len(results), results)
	}
	if !reflect.DeepEqual(expectedResult1, results[0]) && !reflect.DeepEqual(expectedResult1, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult1, results)
	}
	if !reflect.DeepEqual(expectedResult2, results[0]) && !reflect.DeepEqual(expectedResult2, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult2, results)
	}

	// all events with different backoff
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.pushBackOff(expectedResult2)
	queue.pushBackOff(expectedResult2) // increase backoff
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.pushBackOff(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult2, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult2, result)
	}

	// all events with backoff resetted by non-backoff event
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.pushBackOff(expectedResult1)
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.pushBackOff(expectedResult2)
	queue.pushBackOff(expectedResult2) // increase backoff
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult2, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult2, result)
	}
}

func TestEventQueueParallel(t *testing.T) {
	queue := newEventQueue()

	expectedResult1 := newDeviceEvent("sda", "remove")
	expectedResult2 := newDeviceEvent("sdb", "remove")
	var wg sync.WaitGroup
	var results []*deviceEvent
	var resultsMutex sync.Mutex

	push := func(event *deviceEvent) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queue.push(event)
		}()
	}

	pop := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := queue.pop()
			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()
		}()
	}

	push(expectedResult1)
	push(expectedResult2)
	pop()
	pop()

	wg.Wait()

	if len(results) != 2 {
		t.Fatalf("expected: 2, got: %v of %v", len(results), results)
	}

	if !reflect.DeepEqual(expectedResult1, results[0]) && !reflect.DeepEqual(expectedResult1, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult1, results)
	}

	if !reflect.DeepEqual(expectedResult2, results[0]) && !reflect.DeepEqual(expectedResult2, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult2, results)
	}
}

func TestEventQueueBackOffParallel(t *testing.T) {
	queue := newEventQueue()

	expectedResult1 := newDeviceEvent("sda", "remove")
	expectedResult2 := newDeviceEvent("sdb", "remove")
	var wg sync.WaitGroup
	var results []*deviceEvent
	var resultsMutex sync.Mutex

	push := func(event *deviceEvent, increaseBackoff int) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			queue.pushBackOff(event)
			for i := 0; i < increaseBackoff; i++ {
				queue.pushBackOff(event)
			}
		}()
	}

	pop := func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := queue.pop()
			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()
		}()
	}

	push(expectedResult1, 0)
	push(expectedResult2, 2)
	pop()
	pop()

	wg.Wait()

	if len(results) != 2 {
		t.Fatalf("expected: 2, got: %v of %v", len(results), results)
	}

	if !reflect.DeepEqual(expectedResult1, results[0]) && !reflect.DeepEqual(expectedResult1, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult1, results)
	}

	if !reflect.DeepEqual(expectedResult2, results[0]) && !reflect.DeepEqual(expectedResult2, results[1]) {
		t.Fatalf("expected result %v not found in %v", expectedResult2, results)
	}
}

// Just to increase code coverage
func TestTimerString(t *testing.T) {
	timer := timeAfterFunc(1, func() {})
	expectedResult := "timer{1ns}"
	result := timer.String()
	if result != expectedResult {
		t.Fatalf("expected: %v, got: %v", expectedResult, result)
	}
}

// Just to increase code coverage
func TestDeviceEventString(t *testing.T) {
	event := newDeviceEvent("sda", "remove")
	expectedResult := "{created:" + event.created.String() + ", devPath:sda, action:remove, timer:<nil>}"
	result := event.String()
	if result != expectedResult {
		t.Fatalf("expected: %v, got: %v", expectedResult, result)
	}
}
