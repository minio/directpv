/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2021, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package sys

import (
	"context"

	"os/exec"
)

func format(ctx context.Context, path, fs string, options []string, force bool) (string, error) {
	bin := "mkfs." + fs
	args := func() []string {
		args := options
		if force {
			args = append(args, "-f")
		}
		return append(args, path)
	}()

	cmd := exec.CommandContext(ctx, bin, args...)
	outputBytes, err := cmd.CombinedOutput()
	return string(outputBytes), err
}

func setXFSUUID(ctx context.Context, uuid, path string) (string, error) {
	bin := "xfs_admin"
	args := func() []string {
		args := []string{"-U", uuid}
		return append(args, path)
	}()

	cmd := exec.CommandContext(ctx, bin, args...)
	outputBytes, err := cmd.CombinedOutput()
	return string(outputBytes), err
}
