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

package node

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestFindDrives(t *testing.T) {
	ctx := context.Background()
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	_, err = FindDrives(ctx, hostname, "/proc")
	if err != nil {
		t.Error(err)
	}
}

func TestGetLatestStatus(t1 *testing.T) {
	testCases := []struct {
		name           string
		reqStatusList  []metav1.Condition
		selectedStatus metav1.Condition
	}{
		{
			name: "test1",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "staged",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "staged",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test2",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test3",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "test2",
			reqStatusList: []metav1.Condition{
				{
					Type:               "staged",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Date(2000, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					Type:               "published",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			selectedStatus: metav1.Condition{
				Type:               "published",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Date(2000, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range testCases {
		t1.Run(tt.name, func(t1 *testing.T) {
			resStatus := GetLatestStatus(tt.reqStatusList)
			if !reflect.DeepEqual(resStatus, tt.selectedStatus) {
				t1.Errorf("Test case name %s: Expected status option = %v, got %v", tt.name, tt.selectedStatus, resStatus)
			}
		})
	}

}

func TestNSLock(t *testing.T) {
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		nsLk := newNSLockMap()
		resource := "node/path"

		// Scenario-1: Obtain a simple lock on the resource
		if !nsLk.lockLoop(ctx, resource, 3*time.Second, 1*time.Second) {
			t.Fatal("failed to acquire lock")
		}

		// Scenario-2: The locking shoud fail as the resource is already locked
		if nsLk.lock(ctx, resource) {
			t.Fatal("failed to lock the resource")
		}
		nsLk.unlock(resource)

		// Scenario-3: This locking should be successful as the resource is unlocked
		if !nsLk.lock(ctx, resource) {
			t.Fatal("failed to lock the unlocked resource")
		}
		nsLk.unlock(resource)

		// Scenario-4: Re-acquire a released lock
		if !nsLk.lockLoop(ctx, resource, 3*time.Second, 1*time.Millisecond) {
			t.Fatal("failed to re-acquire the lock")
		}

		// Scenario-5: Spawn two parallel threads to obtain a lock
		lk1ch := make(chan bool)
		go func() {
			acquired := nsLk.lockLoop(ctx, resource, 3*time.Second, 1*time.Millisecond)
			lk1ch <- acquired
			nsLk.unlock(resource)
		}()

		lk2ch := make(chan bool)
		go func() {
			acquired := nsLk.lockLoop(ctx, resource, 3*time.Second, 1*time.Millisecond)
			lk2ch <- acquired
			nsLk.unlock(resource)
		}()
		runtime.Gosched()

		// Wait for the lock release signal
		var wg sync.WaitGroup
		var lk1ok, lk2ok bool
		go func() {
			wg.Add(1)
			lk1ok = <-lk1ch
			wg.Done()
		}()

		go func() {
			wg.Add(1)
			lk2ok = <-lk2ch
			wg.Done()
		}()

		// Release the lock being held (re-acquired lock) after 3 millisecs.
		time.Sleep(3 * time.Millisecond)
		nsLk.unlock(resource)

		// wait for the results
		wg.Wait()

		// Both the threads should have obtained the lock sequentially (on-waiting and retrying)
		if !lk1ok || !lk2ok {
			t.Fatalf("Lock conflicts; iteration=%d, lk1=%t, lk2=%t", i, lk1ok, lk2ok)
		}
	}
}
