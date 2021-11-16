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
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

// formatDrive - Idempotent function to format a DirectCSIDrive
func formatDrive(ctx context.Context, uuid, path string, force bool) error {
	output, err := format(ctx, path, string(FSTypeXFS), []string{"-i", "maxpct=50", "-m", "reflink=0"}, force)
	if err != nil {
		klog.Errorf("failed to format drive: %s", output)
		return fmt.Errorf("error while formatting: %v output: %s", err, output)
	}
	if uuid != "" {
		output, err = setXFSUUID(ctx, uuid, path)
		if err != nil {
			klog.Errorf("failed to set uuid after formatting: %s", output)
			return fmt.Errorf("error while setting uuid: %v output: %s", err, output)
		}
	}
	return nil
}

// DriveFormatter denotes filesystem making interface.
type DriveFormatter interface {
	FormatDrive(ctx context.Context, uuid, path string, force bool) error
	MakeBlockFile(path string, major, minor uint32) error
}

// DefaultDriveFormatter is a default filesystem making interface.
type DefaultDriveFormatter struct{}

// FormatDrive makes XFS filesystem on given device.
func (c *DefaultDriveFormatter) FormatDrive(ctx context.Context, uuid, path string, force bool) error {
	return formatDrive(ctx, uuid, path, force)
}

// MakeBlockFile creates device file by it's major/minor number.
func MakeBlockFile(path string, major, minor uint32) error {
	mknod := func(path string, major, minor uint32) error {
		return unix.Mknod(path, unix.S_IFBLK|0666, int(unix.Mkdev(major, minor)))
	}

	if err := mknod(path, major, minor); err == nil || !errors.Is(err, os.ErrExist) {
		return err
	}

	majorN, minorN, err := GetMajorMinor(path)
	if err != nil {
		return err
	}
	if majorN == major && minorN == minor {
		// No change in (major, minor) pair
		return nil
	}

	// remove and a create a new device with correct (major, minor) pair
	if err := os.Remove(path); err != nil {
		return err
	}

	return mknod(path, major, minor)
}

// MakeBlockFile creates device file by it's major/minor number.
func (c *DefaultDriveFormatter) MakeBlockFile(path string, major, minor uint32) error {
	return MakeBlockFile(path, major, minor)
}
