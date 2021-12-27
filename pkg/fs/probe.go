/*
 * This file is part of MinIO Direct CSI
 * Copyright (c) 2021 MinIO, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package fs

import (
	"context"
	"fmt"
	"io"
	"os"

	fserrors "github.com/minio/directpv/pkg/fs/errors"
	"github.com/minio/directpv/pkg/fs/ext4"
	"github.com/minio/directpv/pkg/fs/fat32"
	"github.com/minio/directpv/pkg/fs/swap"
	"github.com/minio/directpv/pkg/fs/xfs"
)

// FS denotes filesystem interface.
type FS interface {
	ID() string
	Type() string
	TotalCapacity() uint64
	FreeCapacity() uint64
}

func probe(device string) (FS, error) {
	devFile, err := os.OpenFile(device, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	xfsSB, err := xfs.Probe(devFile)
	if err == nil {
		return xfsSB, nil
	}

	if _, err = devFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	ext4SB, err := ext4.Probe(devFile)
	if err == nil {
		return ext4SB, nil
	}

	if _, err = devFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	fat32SB, err := fat32.Probe(devFile)
	if err == nil {
		return fat32SB, nil
	}

	if _, err = devFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	swapSB, err := swap.Probe(devFile)
	if err != nil {
		return nil, err
	}

	return swapSB, nil
}

// Probe detects and returns filesystem information of given device.
func Probe(ctx context.Context, device string) (fs FS, err error) {
	doneCh := make(chan struct{})
	go func() {
		fs, err = probe(device)
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w; %v", fserrors.ErrCanceled, ctx.Err())
	case <-doneCh:
	}

	return fs, err
}

func getCapacity(device, filesystem string) (totalCapacity, freeCapacity uint64, err error) {
	var devFile *os.File
	switch filesystem {
	case "xfs", "ext4", "vfat":
		if devFile, err = os.OpenFile(device, os.O_RDONLY, os.ModeDevice); err != nil {
			return 0, 0, err
		}
		defer devFile.Close()
	case "swap":
		return 0, 0, nil
	default:
		return 0, 0, fserrors.ErrFSNotFound
	}

	switch filesystem {
	case "xfs":
		xfsSB, err := xfs.Probe(devFile)
		if err != nil {
			return 0, 0, err
		}
		return xfsSB.TotalCapacity(), xfsSB.FreeCapacity(), nil
	case "ext4":
		ext4SB, err := ext4.Probe(devFile)
		if err != nil {
			return 0, 0, err
		}
		return ext4SB.TotalCapacity(), ext4SB.FreeCapacity(), nil
	case "vfat":
		fat32SB, err := fat32.Probe(devFile)
		if err != nil {
			return 0, 0, err
		}
		return fat32SB.TotalCapacity(), fat32SB.FreeCapacity(), nil
	}

	return 0, 0, fserrors.ErrFSNotFound
}

// GetCapacity returns total/free capacity from filesystem on device.
func GetCapacity(ctx context.Context, device, filesystem string) (totalCapacity, freeCapacity uint64, err error) {
	doneCh := make(chan struct{})
	go func() {
		totalCapacity, freeCapacity, err = getCapacity(device, filesystem)
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return 0, 0, fmt.Errorf("%w; %v", fserrors.ErrCanceled, ctx.Err())
	case <-doneCh:
	}

	return totalCapacity, freeCapacity, err
}
