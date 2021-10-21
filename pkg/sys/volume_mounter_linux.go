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

	"k8s.io/klog/v2"
)

// Idempotent function to bind mount a xfs filesystem with limits
func mountVolume(ctx context.Context, src, dest string, readOnly bool) error {
	klog.V(5).Infof("[mountVolume] source: %v destination: %v", src, dest)
	if err := safeMount(src, dest, string(FSTypeXFS),
		func() []MountOption {
			mOpts := []MountOption{
				MountOptionMSBind,
			}
			if readOnly {
				mOpts = append(mOpts, MountOptionMSReadOnly)
			}
			return mOpts
		}(), []string{quotaOption}); err != nil {
		return err
	}

	return nil
}

func unmountVolume(targetPath string) error {
	return SafeUnmount(targetPath, nil)
}

// VolumeMounter is mount/unmount of volume interface.
type VolumeMounter interface {
	MountVolume(ctx context.Context, src, dest string, readOnly bool) error
	UnmountVolume(targetPath string) error
}

// DefaultVolumeMounter is a default mount/unmount of volume interface.
type DefaultVolumeMounter struct{}

// MountVolume mounts a volume.
func (c *DefaultVolumeMounter) MountVolume(ctx context.Context, src, dest string, readOnly bool) error {
	return mountVolume(ctx, src, dest, readOnly)
}

// UnmountVolume unmounts a volume.
func (c *DefaultVolumeMounter) UnmountVolume(targetPath string) error {
	return unmountVolume(targetPath)
}
