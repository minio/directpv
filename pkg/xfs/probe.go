// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

import "errors"

// MinSupportedDeviceSize is minimum supported size for default XFS filesystem.
const MinSupportedDeviceSize = 16 * 1024 * 1024 // 16 MiB

// ErrFSNotFound denotes filesystem not found error.
var ErrFSNotFound = errors.New("filesystem not found")

// Probe probes XFS filesystem on device.
func Probe(device string) (fsuuid, label string, totalCapacity, freeCapacity uint64, err error) {
	return probe(device)
}
