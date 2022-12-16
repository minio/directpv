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
	"sync/atomic"

	"github.com/minio/directpv/pkg/k8s"
	"k8s.io/klog/v2"
)

// Init initializes legacy clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}

	k8s.Init()

	var err error
	if driveClient, err = DirectCSIDriveInterfaceForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new DirectCSI drive interface; %v", err)
	}

	if volumeClient, err = DirectCSIVolumeInterfaceForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new DirectCSI volume interface; %v", err)
	}
}
