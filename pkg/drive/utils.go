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
	"context"
	"fmt"
	"os"

	"github.com/minio/direct-csi/pkg/sys"

	"github.com/golang/glog"
)

// mountDrive - Idempotent function to mount a DirectCSIDrive
func mountDrive(source, target string, mountOpts []string) error {
	// Since pods will be consuming this target, be permissive
	if err := os.MkdirAll(target, 0777); err != nil {
		return err
	}

	glog.V(3).Infof("mounting drive %s at %s", source, target)
	return sys.SafeMount(source, target, string(sys.FSTypeXFS), func(opts []string) []sys.MountOption {
		newOpts := []sys.MountOption{}
		for _, opt := range opts {
			newOpts = append(newOpts, sys.MountOption(opt))
		}
		return newOpts
	}(mountOpts), []string{
		"prjquota",
	})

	return nil
}

// unmountDrive - Idempotent function to unmount a DirectCSIDrive
func unmountDrive(drivePath string) error {
	glog.V(3).Infof("unmounting drive %s", drivePath)
	if err := sys.SafeUnmountAll(drivePath, []sys.UnmountOption{
		sys.UnmountOptionDetach,
		sys.UnmountOptionForce,
	}); err != nil {
		return err
	}

	return nil
}

// formatDrive - Idempotent function to format a DirectCSIDrive
func formatDrive(ctx context.Context, path string, force bool) error {
	output, err := sys.Format(ctx, path, string(sys.FSTypeXFS), force)
	if err != nil {
		glog.Errorf("failed to format drive: %s", output)
		return fmt.Errorf("%s", output)
	}
	return nil
}
