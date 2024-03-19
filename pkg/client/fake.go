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

package client

import (
	"github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
)

// FakeInit initializes fake clients.
func FakeInit() {
	k8s.FakeInit()
	k8sClient := k8s.GetClient()
	clientsetInterface := types.NewExtFakeClientset(fake.NewSimpleClientset())
	driveClient := clientsetInterface.DirectpvLatest().DirectPVDrives()
	volumeClient := clientsetInterface.DirectpvLatest().DirectPVVolumes()
	nodeClient := clientsetInterface.DirectpvLatest().DirectPVNodes()
	initRequestClient := clientsetInterface.DirectpvLatest().DirectPVInitRequests()
	restClient := clientsetInterface.DirectpvLatest().RESTClient()

	initEvent(k8sClient.KubeClient)
	client = &Client{
		K8sClient:          k8sClient,
		ClientsetInterface: clientsetInterface,
		RESTClient:         restClient,
		DriveClient:        driveClient,
		VolumeClient:       volumeClient,
		NodeClient:         nodeClient,
		InitRequestClient:  initRequestClient,
	}
}

// SetDriveInterface sets latest drive interface.
// Note: To be used for writing test cases only
func SetDriveInterface(i types.LatestDriveInterface) {
	client.DriveClient = i
}

// SetVolumeInterface sets the latest volume interface.
// Note: To be used for writing test cases only
func SetVolumeInterface(i types.LatestVolumeInterface) {
	client.VolumeClient = i
}

// SetNodeInterface sets latest node interface.
// Note: To be used for writing test cases only
func SetNodeInterface(i types.LatestNodeInterface) {
	client.NodeClient = i
}

// SetInitRequestInterface sets latest initrequest interface.
// Note: To be used for writing test cases only
func SetInitRequestInterface(i types.LatestInitRequestInterface) {
	client.InitRequestClient = i
}
