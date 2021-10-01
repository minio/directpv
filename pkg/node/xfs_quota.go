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

package node

import (
	"context"

	"github.com/minio/direct-csi/pkg/fs/xfs"
)

type quotaFuncs interface {
	GetQuota(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error)
	SetQuota(ctx context.Context, device, path, volumeID string, quota xfs.Quota) (err error)
}

type xfsQuotaFuncs struct{}

func (q *xfsQuotaFuncs) GetQuota(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error) {
	return xfs.GetQuota(ctx, device, volumeID)
}

func (q *xfsQuotaFuncs) SetQuota(ctx context.Context, device, path, volumeID string, quota xfs.Quota) (err error) {
	return xfs.SetQuota(ctx, device, path, volumeID, quota)
}
