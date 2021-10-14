//go:build !linux

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

package sys

import (
	"fmt"
	"runtime"

	"k8s.io/klog/v2"
)

func safeMount(source, target, fsType string, mountOpts []MountOption, superblockOpts []string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func mount(source, target, fsType string, mountOpts []MountOption, superblockOpts []string) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func SafeUnmount(target string, opts []UnmountOption) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func safeUnmountAll(path string, opts []UnmountOption) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func Unmount(target string, opts []UnmountOption) error {
	return fmt.Errorf("unsupported operating system %v", runtime.GOOS)
}

func ForceUnmount(target string) {
	klog.V(5).Infof("unsupported operating system %v", runtime.GOOS)
}
