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
	"sync/atomic"

	"github.com/minio/directpv/pkg/clientset"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"k8s.io/klog/v2"
)

// Init initializes various clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}

	k8s.Init()

	cs, err := clientset.NewForConfig(k8s.KubeConfig())
	if err != nil {
		klog.Fatalf("unable to create new clientset interface; %v", err)
	}
	clientsetInterface = types.NewExtClientset(cs)

	restClient = clientsetInterface.DirectpvLatest().RESTClient()

	if driveClient, err = latestDriveClientForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new drive interface; %v", err)
	}

	if volumeClient, err = latestVolumeClientForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new volume interface; %v", err)
	}

	if nodeClient, err = latestNodeClientForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new node interface; %v", err)
	}

	if initRequestClient, err = latestInitRequestClientForConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to create new initrequest interface; %v", err)
	}

	initEvent(k8s.KubeClient())
}
