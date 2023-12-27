// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package admin

import (
	"context"

	"github.com/minio/directpv/pkg/installer"
)

// UninstallArgs represents the args to uninstall
type UninstallArgs struct {
	Quiet     bool
	Dangerous bool
}

// Uninstall uninstalls directpv
func Uninstall(ctx context.Context, args UninstallArgs) error {
	return installer.Uninstall(ctx, args.Quiet, args.Dangerous)
}
