//go:build linux

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

package xfs

import (
	"context"
	"fmt"
	"os/exec"
)

func makeFS(ctx context.Context, device, uuid string, force, reflink bool) error {
	args := []string{"-i", "maxpct=50", "-m", fmt.Sprintf("uuid=%v", uuid)}
	if !reflink {
		args = append(args, "-m", "reflink=0")
	}
	if force {
		args = append(args, "-f")
	}
	args = append(args, "-L", "DIRECTCSI", device)

	if output, err := exec.CommandContext(ctx, "mkfs.xfs", args...).CombinedOutput(); err != nil {
		return fmt.Errorf(
			"unable to execute command %v; output=%v; error=%w",
			append([]string{"mkfs.xfs"}, args...), string(output), err,
		)
	}

	return nil
}
