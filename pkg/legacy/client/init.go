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
	"fmt"
	"sync/atomic"

	"github.com/minio/directpv/pkg/k8s"
	"k8s.io/klog/v2"
)

// Init initializes legacy clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}
	var err error
	if err = k8s.Init(); err != nil {
		klog.Fatalf("unable to initialize k8s clients; %v", err)
	}
	client, err = NewClient(k8s.GetClient())
	if err != nil {
		klog.Fatalf("unable to create legacy client; %v", err)
	}
}

// NewClient creates a legacy client
func NewClient(k8sClient *k8s.Client) (*Client, error) {
	driveClient, err := DirectCSIDriveInterfaceForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new DirectCSI drive interface; %w", err)
	}
	volumeClient, err := DirectCSIVolumeInterfaceForConfig(k8sClient)
	if err != nil {
		return nil, fmt.Errorf("unable to create new DirectCSI volume interface; %w", err)
	}
	return &Client{
		DriveClient:  driveClient,
		VolumeClient: volumeClient,
		K8sClient:    k8sClient,
	}, nil
}
