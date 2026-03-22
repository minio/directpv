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

import (
	"context"
	"errors"
)

// ErrCanceled denotes canceled by context error.
var ErrCanceled = errors.New("canceled by context")

// Quota denotes XFS quota information.
type Quota struct {
	HardLimit    uint64
	SoftLimit    uint64
	CurrentSpace uint64
}

// GetQuota returns XFS quota information of given volume ID.
func GetQuota(ctx context.Context, device, volumeID string) (quota *Quota, err error) {
	doneCh := make(chan struct{})
	go func() {
		quota, err = getQuota(device, volumeID)
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return nil, errors.Join(ErrCanceled, ctx.Err())
	case <-doneCh:
	}

	return quota, err
}

// SetQuota sets quota information on given path and volume ID.
func SetQuota(ctx context.Context, device, path, volumeID string, quota Quota, update bool) (err error) {
	doneCh := make(chan struct{})
	go func() {
		err = setQuota(device, path, volumeID, quota, update)
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return errors.Join(ErrCanceled, ctx.Err())
	case <-doneCh:
	}

	return err
}
