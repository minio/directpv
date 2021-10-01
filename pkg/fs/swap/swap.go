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

package swap

import (
	"bytes"
	"errors"
	"io"

	fserrors "github.com/minio/direct-csi/pkg/fs/errors"
)

const (
	swapSignature     = "SWAPSPACE2"
	swapSignatureSize = len(swapSignature) - 1
	maxPageSize       = 64 * 1024
)

type Swap struct{}

func (swap *Swap) ID() string {
	return ""
}

func (swap *Swap) Type() string {
	return "linux-swap"
}

func (swap *Swap) TotalCapacity() uint64 {
	return 0
}

func (swap *Swap) FreeCapacity() uint64 {
	return 0
}

func Probe(reader io.Reader) (*Swap, error) {
	// Refer https://github.com/karelzak/util-linux/blob/master/sys-utils/swapon.c#L426
	// for more information for this logic.
	// The smallest swap area is PAGE_SIZE*10, it means 40k, that's less than maxPageSize
	data := make([]byte, maxPageSize)
	n, err := io.ReadFull(reader, data)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	data = data[:n]

	for page := 0x1000; page <= maxPageSize; page <<= 1 {
		// Skip 32k pagesize since this does not seem to be supported
		if page == 0x8000 {
			continue
		}

		offset := page - swapSignatureSize - 1
		if len(data) < offset {
			break
		}

		if bytes.HasPrefix(data[offset:], []byte(swapSignature)) {
			// TODO: read swap header for UUID and volume name.
			// Refer https://github.com/karelzak/util-linux/blob/master/include/swapheader.h
			return &Swap{}, nil
		}
	}

	return nil, fserrors.ErrFSNotFound
}
