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

package drive

import (
	"testing"
)

func TestVolumeLock(t *testing.T) {

	path := "node/path"
	for i := 0; i < 100; i++ {

		acquireLock(path)

		lk1ch := make(chan bool)
		go func() {
			acquireLock(path)
			lk1ch <- true
			releaseLock(path)
		}()

		lk2ch := make(chan bool)
		go func() {
			acquireLock(path)
			lk2ch <- true
			releaseLock(path)
		}()

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
		runtime.Gosched()

		if lk1ok || lk2ok {
			t.Fatalf("Able to pick a locked resource; iteration=%d, lk1=%t, lk2=%t", i, lk1ok, lk2ok)
		}

		time.Sleep(3 * time.Millisecond)
		releaseLock(path)

		// wait for the results
		wg.Wait()

		if !lk1ok || !lk2ok {
			t.Fatalf("Unable to pick an unlocked resource; iteration=%d, lk1=%t, lk2=%t", i, lk1ok, lk2ok)
		}

	}

	if _, ok := volumeLocker[path]; ok {
		t.Fatal("Failed to clean up the volume locker")
	}
}
