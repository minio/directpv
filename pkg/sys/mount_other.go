//go:build !linux

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

package sys

import (
	"fmt"
	"runtime"
)

func getMounts(proc1Mountinfo string) (mountPointMap, deviceMap map[string][]string, err error) {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func mount(proc1Mountinfo, device, target, fsType string, flags []string, superBlockFlags string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func unmount(proc1Mountinfo, target string, force, detach, expire bool) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func bindMount(proc1Mountinfo, source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}
