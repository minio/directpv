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

	fakedirect "github.com/minio/direct-csi/pkg/clientset/fake"
	"github.com/minio/direct-csi/pkg/sys/fs/quota"
)

const (
	testNodeName = "test-node"
)

type fakeVolumeMounter struct {
	mountArgs struct {
		source      string
		destination string
		readOnly    bool
	}
	unmountArgs struct {
		target string
	}
}

func (f *fakeVolumeMounter) MountVolume(_ context.Context, src, dest string, readOnly bool) error {
	f.mountArgs.source = src
	f.mountArgs.destination = dest
	f.mountArgs.readOnly = readOnly
	return nil
}

func (f *fakeVolumeMounter) UnmountVolume(targetPath string) error {
	f.unmountArgs.target = targetPath
	return nil
}

func createFakeNodeServer() *NodeServer {
	return &NodeServer{
		NodeID:          testNodeName,
		Identity:        "test-identity",
		Rack:            "test-rack",
		Zone:            "test-zone",
		Region:          "test-region",
		directcsiClient: fakedirect.NewSimpleClientset(),
		mounter:         &fakeVolumeMounter{},
		quotaer:         &quota.FakeDriveQuotaer{},
	}
}
