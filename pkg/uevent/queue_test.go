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

func checkResults(t *testing.T, results []*deviceEvent, expectedResults ...*deviceEvent) {
	if len(results) != len(expectedResults) {
		t.Fatalf("results count: expected: %v, got: %v", len(expectedResults), len(results))
	}

	for _, expectedResult := range expectedResults {
		found := false
		for _, result := range results {
			if found = reflect.DeepEqual(expectedResult, result); found {
				break
			}
		}

		if !found {
			t.Fatalf("expected result %v not found in %v", expectedResult, results)
		}
	}
}

func TestEventQueue(t *testing.T) {
	queue := newEventQueue()

	var expectedResult1, expectedResult2, result *deviceEvent
	var results []*deviceEvent

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
	results = []*deviceEvent{
		queue.pop(),
		queue.pop(),
	}
	checkResults(t, results, expectedResult1, expectedResult2)

	// with/without backoff pushes and second pop returns backoff pushed event
	expectedResult2 = newDeviceEvent("sda", "change")
	queue.push(expectedResult2)
	expectedResult1 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult1)
	results = []*deviceEvent{
		queue.pop(),
		queue.pop(),
	}
	checkResults(t, results, expectedResult1, expectedResult2)

	// the latest event replaces backoff push
	expectedResult1 = newDeviceEvent("sda", "change")
	queue.push(expectedResult1)
	expectedResult1 = queue.pop()
	queue.push(expectedResult1) // backoff added
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// the latest event with backoff replaces older event
	queue.push(newDeviceEvent("sda", "change"))
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	expectedResult1 = queue.pop()
	queue.push(expectedResult1)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}

	// the latest event is not replaced by older event
	oldEvent := newDeviceEvent("sda", "change")
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	queue.push(oldEvent)
	result = queue.pop()
	if !reflect.DeepEqual(expectedResult1, result) {
		t.Fatalf("expected: %v, got: %v", expectedResult1, result)
	}
}

func TestEventQueueBackOff(t *testing.T) {
	queue := newEventQueue()

	var expectedResult1, expectedResult2 *deviceEvent
	var results []*deviceEvent

	// all events with backoff
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	expectedResult1 = queue.pop()
	queue.push(expectedResult1) // backoff added
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult2)
	expectedResult2 = queue.pop()
	queue.push(expectedResult2) // backoff added
	results = []*deviceEvent{
		queue.pop(),
		queue.pop(),
	}
	checkResults(t, results, expectedResult1, expectedResult2)

	// all events with different backoff
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	expectedResult1 = queue.pop()
	queue.push(expectedResult1) // backoff added
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult2)
	expectedResult2 = queue.pop()
	queue.push(expectedResult2) // backoff added
	expectedResult2 = queue.pop()
	queue.push(expectedResult2) // backoff added
	results = []*deviceEvent{
		queue.pop(),
		queue.pop(),
	}
	checkResults(t, results, expectedResult1, expectedResult2)

	// all events with backoff resetted by new event
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	expectedResult1 = queue.pop()
	queue.push(expectedResult1) // backoff added
	expectedResult2 = newDeviceEvent("sdb", "remove")
	queue.push(expectedResult2)
	expectedResult2 = queue.pop()
	queue.push(expectedResult2) // backoff added
	expectedResult2 = queue.pop()
	queue.push(expectedResult2) // backoff added
	expectedResult1 = newDeviceEvent("sda", "remove")
	queue.push(expectedResult1)
	results = []*deviceEvent{
		queue.pop(),
		queue.pop(),
	}
	checkResults(t, results, expectedResult1, expectedResult2)
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

	checkResults(t, results, expectedResult1, expectedResult2)
}

func TestEventQueueBackOffParallel(t *testing.T) {
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
			event = queue.pop()
			queue.push(event) // backoff added
		}()
	}
	push(expectedResult1)
	push(expectedResult2)
	wg.Wait()

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
	pop()
	pop()
	wg.Wait()

	checkResults(t, results, expectedResult1, expectedResult2)
}
