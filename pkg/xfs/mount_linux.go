//go:build linux

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

package xfs

import (
	"errors"
	"os"
	"path"

	"github.com/minio/directpv/pkg/sys"
	"k8s.io/klog/v2"
)

func mount(device, target string) error {
	if err := sys.Mkdir(target, 0o777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	if err := sys.Mount(device, target, "xfs", []string{"noatime"}, "prjquota"); err != nil {
		return err
	}

	name := path.Base(device)
	if name == "/" || name == "." {
		klog.Errorf("unable to get device name from device %v", device)
		return nil
	}

	if err := os.WriteFile("/sys/fs/xfs/"+name+"/error/metadata/EIO/max_retries", []byte("1"), 0o644); err != nil {
		klog.ErrorS(err, "unable to set EIO max_retires device", "name", name)
	}

	if err := os.WriteFile("/sys/fs/xfs/"+name+"/error/metadata/EIO/retry_timeout_seconds", []byte("5"), 0o644); err != nil {
		klog.ErrorS(err, "unable to set EIO retry_timeout_seconds for device", "name", name)
	}

	if err := os.WriteFile("/sys/fs/xfs/"+name+"/error/metadata/ENOSPC/max_retries", []byte("1"), 0o644); err != nil {
		klog.ErrorS(err, "unable to set ENOSPC max_retires device", "name", name)
	}

	if err := os.WriteFile("/sys/fs/xfs/"+name+"/error/metadata/ENOSPC/retry_timeout_seconds", []byte("5"), 0o644); err != nil {
		klog.ErrorS(err, "unable to set ENOSPC retry_timeout_seconds for device", "name", name)
	}

	return nil
}

func bindMount(source, target string, readOnly bool) error {
	return sys.BindMount(source, target, "xfs", false, readOnly, "prjquota")
}
