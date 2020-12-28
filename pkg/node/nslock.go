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
	"sync"
	"time"
)

type nsLockMap struct {
	lockMap      map[string]bool
	lockMapMutex sync.Mutex
}

// newNSLock - return a new name space lock map.
func newNSLockMap() *nsLockMap {
	nsMutex := nsLockMap{}
	nsMutex.lockMap = make(map[string]bool)
	return &nsMutex
}

// Wait and acquire loop with timeout and retry
func (n *nsLockMap) lockLoop(ctx context.Context, resource string, timeout, retryInterval time.Duration) (locked bool) {
	retryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-retryCtx.Done():
			// Caller context canceled or we timedout,
			// return false anyways for both situations.
			return false
		default:
			if n.lock(ctx, resource) {
				return true
			}
			time.Sleep(retryInterval)
		}
	}
}

// Lock the namespace resource.
func (n *nsLockMap) lock(ctx context.Context, resource string) bool {
	n.lockMapMutex.Lock()
	defer n.lockMapMutex.Unlock()

	// Check if the resource is locked
	isLocked, found := n.lockMap[resource]
	if !found {
		n.lockMap[resource] = true
		return true
	}

	// Fail to obtain a lock
	if isLocked {
		return false
	}

	// Resource is free. Lock it.
	n.lockMap[resource] = true
	return true
}

// Unlock the namespace resource.
func (n *nsLockMap) unlock(resource string) {
	n.lockMapMutex.Lock()
	defer n.lockMapMutex.Unlock()
	if _, found := n.lockMap[resource]; !found {
		return
	}
	delete(n.lockMap, resource)
	return
}
