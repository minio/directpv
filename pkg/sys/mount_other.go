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

	"github.com/minio/directpv/pkg/utils"
)

func getMounts(includeMajorMinorMap bool) (mountPointMap, deviceMap, majorMinorMap, rootMountPointMap map[string]utils.StringSet, err error) {
	return nil, nil, nil, nil, fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func mount(device, target, fsType string, flags []string, superBlockFlags string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func unmount(target string, force, detach, expire bool) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func bindMount(source, target, fsType string, recursive, readOnly bool, superBlockFlags string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}
