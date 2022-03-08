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

	directsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/fs"
	"github.com/minio/directpv/pkg/fs/xfs"
	"github.com/minio/directpv/pkg/mount"
)

const (
	testNodeName = "test-node"
)

type fakeXFS struct{}

func (fxfs *fakeXFS) ID() string {
	return ""
}
func (fxfs *fakeXFS) Type() string {
	return "xfs"
}
func (fxfs *fakeXFS) TotalCapacity() uint64 {
	return uint64(0)
}
func (fxfs *fakeXFS) FreeCapacity() uint64 {
	return uint64(0)
}

func createFakeNodeServer() *NodeServer {
	return &NodeServer{
		NodeID:          testNodeName,
		Identity:        "test-identity",
		Rack:            "test-rack",
		Zone:            "test-zone",
		Region:          "test-region",
		directcsiClient: directsetfake.NewSimpleClientset(),
		probeMounts: func() (map[string][]mount.MountInfo, error) {
			return map[string][]mount.MountInfo{"0:0": {{MountPoint: "/var/lib/direct-csi/mnt"}}}, nil
		},
		getDevice:     func(major, minor uint32) (string, error) { return "", nil },
		safeBindMount: func(source, target string, recursive, readOnly bool) error { return nil },
		safeUnmount:   func(target string, force, detach, expire bool) error { return nil },
		getQuota: func(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error) {
			return &xfs.Quota{}, nil
		},
		setQuota: func(ctx context.Context, device, path, volumeID string, quota xfs.Quota) (err error) { return nil },
		fsProbe: func(ctx context.Context, device string) (fs fs.FS, err error) {
			return &fakeXFS{}, nil
		},
	}
}
