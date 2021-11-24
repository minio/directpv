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
	"reflect"
	"testing"
	"time"
)

func TestListenerGet(t *testing.T) {
	events := map[string]*Event{
		"/mnt/a": {
			mountInfo: Info{
				MajorMinor:   "0:100",
				MountOptions: []string{"opt1", "opt2"},
			},
			eventType: Attached,
		},
		"/mnt/b": {
			mountInfo: Info{
				MajorMinor:   "0:101",
				MountOptions: []string{"opt3", "opt4"},
			},
			eventType: Modified,
		},
		"/mnt/c": {
			mountInfo: Info{
				MajorMinor:   "0:102",
				MountOptions: []string{"opt3", "opt4"},
			},
			eventType: Detached,
		},
	}

	testListener := &Listener{
		closeCh:  make(chan struct{}),
		eventMap: make(map[string]*Event),
	}

	for k, v := range events {
		testListener.set(k, v)
		key, event, err := testListener.Get(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
		if key != k {
			t.Fatalf("expected key %s but got %s", k, key)
		}
		if !reflect.DeepEqual(v, event) {
			t.Fatalf("expected event %v but got %v", v, event)
		}
	}

	if len(testListener.eventMap) != 0 {
		t.Fatalf("expected empty eventMap, but got %v", testListener.eventMap)
	}
}

func TestListenerProcess(t *testing.T) {
	testCases := []struct {
		info             info
		expectedEventMap map[string]*Event
	}{
		// Mounts unchanged
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{},
		},
		// New mounts attached
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
					"0:101": {
						{
							MountPoint:   "/mnt/d",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{
				"/mnt/d": {
					mountInfo: Info{
						MountPoint:   "/mnt/d",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Attached,
				},
			},
		},
		// Mounts attached to existing mountlist
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/d",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{
				"/mnt/d": {
					mountInfo: Info{
						MountPoint:   "/mnt/d",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Attached,
				},
			},
		},
		// Mounts detached
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
					"0:101": {
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{
				"/mnt/c": {
					mountInfo: Info{
						MountPoint:   "/mnt/c",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Detached,
				},
			},
		},
		// Mounts detached from existing mount list
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{
				"/mnt/c": {
					mountInfo: Info{
						MountPoint:   "/mnt/c",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Detached,
				},
			},
		},
		// mounts detached and attached
		{
			info: info{
				currentMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/c",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
				previousMountInfo: map[string][]Info{
					"0:100": {
						{
							MountPoint:   "/mnt/a",
							MountOptions: []string{"opt1", "opt2"},
						},
						{
							MountPoint:   "/mnt/b",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
					"0:101": {
						{
							MountPoint:   "/mnt/d",
							MountOptions: []string{"opt1", "opt2"},
						},
					},
				},
			},
			expectedEventMap: map[string]*Event{
				"/mnt/c": {
					mountInfo: Info{
						MountPoint:   "/mnt/c",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Attached,
				},
				"/mnt/d": {
					mountInfo: Info{
						MountPoint:   "/mnt/d",
						MountOptions: []string{"opt1", "opt2"},
					},
					eventType: Detached,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	testListener := &Listener{
		closeCh:  make(chan struct{}),
		eventMap: make(map[string]*Event),
		infoCh:   make(chan info, 16),
	}
	testListener.process(ctx)

	for i, testCase := range testCases {
		select {
		case <-ctx.Done():
			t.Fatal("context canceled")
		default:
			testListener.infoCh <- info{
				previousMountInfo: testCase.info.previousMountInfo,
				currentMountInfo:  testCase.info.currentMountInfo,
			}
		}
		expectedEventCount := len(testCase.expectedEventMap)
		eventMap := make(map[string]*Event)
		for c := 0; c < expectedEventCount; c++ {
			key, event, err := testListener.Get(ctx)
			if err != nil {
				t.Fatal(err)
			}
			eventMap[key] = event
		}
		if !reflect.DeepEqual(eventMap, testCase.expectedEventMap) {
			t.Fatalf("case %v: expected eventMap: %v but got: %v", i, testCase.expectedEventMap, eventMap)
		}
	}
	if len(testListener.eventMap) != 0 {
		t.Fatalf("expected empty eventMap, but got %v", testListener.eventMap)
	}
}
