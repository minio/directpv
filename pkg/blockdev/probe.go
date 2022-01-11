/*
 * This file is part of MinIO Direct PV
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

package blockdev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/minio/directpv/pkg/blockdev/gpt"
	"github.com/minio/directpv/pkg/blockdev/mbr"
	"github.com/minio/directpv/pkg/blockdev/parttable"
)

func probe(devFile *os.File) (parttable.PartTable, error) {
	mbrPT, err := mbr.Probe(devFile)
	if err == nil {
		return mbrPT, nil
	}
	if !errors.Is(err, parttable.ErrPartTableNotFound) && !errors.Is(err, mbr.ErrGPTProtectiveMBR) {
		return nil, err
	}

	if _, err = devFile.Seek(512, io.SeekStart); err != nil {
		return nil, err
	}

	gptPT, err := gpt.Probe(devFile)
	if err != nil {
		return nil, err
	}
	return gptPT, nil
}

// Probe detects and returns partition table in given device device.
func Probe(ctx context.Context, device string) (partTable parttable.PartTable, err error) {
	devFile, err := os.OpenFile(device, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	doneCh := make(chan struct{})
	go func() {
		partTable, err = probe(devFile)
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w; %v", parttable.ErrCancelled, ctx.Err())
	case <-doneCh:
	}

	return partTable, err
}
