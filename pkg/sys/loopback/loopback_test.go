//go:build !skip
// +build !skip

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

package loopback

import (
	"fmt"
	"testing"
	"time"
)

func TestCreateLoopbackDevice(t1 *testing.T) {

	loopPath, bErr := CreateLoopbackDevice()
	if bErr != nil {
		t1.Errorf("Cannot create fake loop device: %v", bErr)
	}
	fmt.Printf("Allocated Loop device: %v\n", loopPath)

	time.Sleep(5 * time.Second)
	if err := RemoveLoopDevice(loopPath); err != nil {
		t1.Errorf("Cannot remove fake loop device: %v", err)
	}
	fmt.Printf("Cleaned up Loop device: %v\n", loopPath)

}
