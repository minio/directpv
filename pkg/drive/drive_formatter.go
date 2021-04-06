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

package drive

import (
	"context"
	"github.com/minio/direct-csi/pkg/utils"
)

type DriveFormatter interface {
	FormatDrive(ctx context.Context, path string, force bool) error
}

type driveFormatter struct{}

func (c *driveFormatter) FormatDrive(ctx context.Context, path string, force bool) error {
	return formatDrive(ctx, path, force)
}

type fakeDriveFormatter struct{}

func (c *fakeDriveFormatter) FormatDrive(ctx context.Context, path string, force bool) error {
	return nil
}

func GetDriveFormatter() DriveFormatter {
	if utils.GetFake() {
		return &fakeDriveFormatter{}
	}
	return &driveFormatter{}
}
