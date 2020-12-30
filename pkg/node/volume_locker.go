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
	"sync"
)

type volumeLock struct {
	ref int32
	m   *sync.Mutex
}

var (
	volumeLocker map[string]*volumeLock
	metaLocker   sync.Mutex
)

func acquireLock(path string) {
	locker := getLocker(path)
	locker.m.Lock()
	return
}

func getLocker(path string) *volumeLock {
	metaLocker.Lock()
	defer metaLocker.Unlock()

	if volumeLocker == nil {
		volumeLocker = make(map[string]*volumeLock)
	}

	lock, ok := volumeLocker[path]
	if !ok {
		volumeLocker[path] = &volumeLock{
			ref: 0,
			m:   &sync.Mutex{},
		}
		lock = volumeLocker[path]
	}
	lock.ref++
	return lock
}

func releaseLock(path string) {
	metaLocker.Lock()
	defer metaLocker.Unlock()

	lock, ok := volumeLocker[path]
	if !ok {
		return
	}
	lock.m.Unlock()
	lock.ref--
	if lock.ref == 0 {
		delete(volumeLocker, path)
	}
	return
}
