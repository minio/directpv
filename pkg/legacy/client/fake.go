// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"github.com/minio/directpv/pkg/k8s"
	legacyclientsetfake "github.com/minio/directpv/pkg/legacy/clientset/fake"
)

// FakeInit initializes fake clients.
func FakeInit() {
	k8s.FakeInit()

	fakeClientset := legacyclientsetfake.NewSimpleClientset()
	driveClient = fakeClientset.DirectV1beta5().DirectCSIDrives()
	volumeClient = fakeClientset.DirectV1beta5().DirectCSIVolumes()
}

// SetDriveClient sets drive interface from fake clientset.
// Note: To be used for writing test cases only
func SetDriveClient(clientset *legacyclientsetfake.Clientset) {
	driveClient = clientset.DirectV1beta5().DirectCSIDrives()
}

// SetVolumeClient sets volume interface from fake clientset.
// Note: To be used for writing test cases only
func SetVolumeClient(clientset *legacyclientsetfake.Clientset) {
	volumeClient = clientset.DirectV1beta5().DirectCSIVolumes()
}
