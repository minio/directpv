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

package fat32

import (
	"fmt"
	"testing"
)

func TestFAT32(t1 *testing.T) {
	f32 := NewFAT32()
	is, err := f32.ProbeFS("/dev/sda1", int64(1048576))
	if err != nil {
		t1.Errorf("Cannot probe: %v", err)
	}
	fmt.Println(is)
}
