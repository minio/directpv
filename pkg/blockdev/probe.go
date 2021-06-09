/*
 * This file is part of MinIO Direct CSI
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
	"errors"
	"os"

	"github.com/minio/direct-csi/pkg/blockdev/gpt"
	"github.com/minio/direct-csi/pkg/blockdev/mbr"
	"github.com/minio/direct-csi/pkg/blockdev/parttable"
)

// Probe detects and returns partition table in given device filename.
func Probe(filename string) (parttable.PartTable, error) {
	devFile, err := os.OpenFile(filename, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return nil, err
	}
	defer devFile.Close()

	mbrPT, err := mbr.Probe(devFile)
	if err == nil {
		return mbrPT, nil
	}
	if !errors.Is(err, parttable.ErrPartTableNotFound) && !errors.Is(err, mbr.ErrGPTProtectiveMBR) {
		return nil, err
	}

	if _, err = devFile.Seek(512, os.SEEK_SET); err != nil {
		return nil, err
	}

	gptPT, err := gpt.Probe(devFile)
	if err != nil {
		return nil, err
	}
	return gptPT, nil
}
