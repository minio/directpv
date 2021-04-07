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
	"fmt"

	"github.com/golang/glog"
)

// formatDrive - Idempotent function to format a DirectCSIDrive
func formatDrive(ctx context.Context, path string, force bool) error {
	output, err := Format(ctx, path, string(FSTypeXFS), []string{"-i", "maxpct=50"}, force)
	if err != nil {
		glog.Errorf("failed to format drive: %s", output)
		return fmt.Errorf("%s", output)
	}
	return nil
}

type DriveFormatter interface {
	FormatDrive(ctx context.Context, path string, force bool) error
}

type DefaultDriveFormatter struct{}

func (c *DefaultDriveFormatter) FormatDrive(ctx context.Context, path string, force bool) error {
	return formatDrive(ctx, path, force)
}
