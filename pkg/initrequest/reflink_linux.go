//go:build linux

// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package initrequest

import (
	"context"
	"errors"
	"os"

	losetup "github.com/freddierice/go-losetup/v2"
	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/xfs"
	"k8s.io/klog/v2"
)

func reflinkSupported(ctx context.Context) (bool, error) {
	errMountFailed := errors.New("unable to mount")

	checkXFS := func(ctx context.Context, reflink bool) error {
		mountPoint, err := os.MkdirTemp("", "xfs.check.mnt.")
		if err != nil {
			return err
		}
		defer os.Remove(mountPoint)

		file, err := os.CreateTemp("", "xfs.check.file.")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		file.Close()

		if err = os.Truncate(file.Name(), xfs.MinSupportedDeviceSize); err != nil {
			return err
		}

		if _, _, _, _, err = xfs.MakeFS(ctx, file.Name(), uuid.New().String(), false, reflink); err != nil {
			return err
		}

		loopDevice, err := losetup.Attach(file.Name(), 0, false)
		if err != nil {
			return err
		}

		defer func() {
			if err := loopDevice.Detach(); err != nil {
				klog.Error(err)
			}
		}()

		if err = xfs.Mount(loopDevice.Path(), mountPoint); err != nil {
			return errors.Join(errMountFailed, err)
		}

		return sys.Unmount(mountPoint, true, true, false)
	}

	reflinkSupport := true
	err := checkXFS(ctx, reflinkSupport)
	if err == nil {
		return reflinkSupport, nil
	}

	if !errors.Is(err, errMountFailed) {
		return false, err
	}

	reflinkSupport = false
	return reflinkSupport, checkXFS(ctx, reflinkSupport)
}
