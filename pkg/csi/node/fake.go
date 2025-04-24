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

package node

import (
	"context"
	"errors"

	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/xfs"
)

const testNodeName = "test-node"

func createFakeServer() *Server {
	return &Server{
		nodeID:            testNodeName,
		identity:          "test-identity",
		rack:              "test-rack",
		zone:              "test-zone",
		region:            "test-region",
		getMounts:         func() (*sys.MountInfo, error) { return nil, nil },
		getDeviceByFSUUID: func(_ string) (string, error) { return "", nil },
		bindMount:         func(_, _ string, _ bool) error { return nil },
		unmount:           func(_ string) error { return nil },
		getQuota: func(_ context.Context, _, _ string) (quota *xfs.Quota, err error) {
			return &xfs.Quota{}, nil
		},
		setQuota: func(_ context.Context, _, _, _ string, _ xfs.Quota, _ bool) (err error) {
			return nil
		},
		mkdir: func(path string) error {
			if path == "" {
				return errors.New("path is empty")
			}
			return nil
		},
	}
}
