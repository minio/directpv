//go:build linux

// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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
	"context"
	"fmt"
	"io"
	"os/exec"
)

func repair(ctx context.Context, device string, force, disablePrefetch, dryRun bool, output io.Writer) error {
	args := []string{device, "-v"}
	if force {
		args = append(args, "-L")
	}
	if disablePrefetch {
		args = append(args, "-P")
	}
	if dryRun {
		args = append(args, "-n")
	}

	cmd := exec.CommandContext(ctx, "xfs_repair", args...)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to run xfs_repair on device %v; %w", device, err)
	}

	return nil
}
