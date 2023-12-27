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
	"fmt"
	"log"
	"sync/atomic"

	"github.com/minio/directpv/pkg/clientset"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// Init initializes various clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}
	if err := k8s.Init(); err != nil {
		log.Fatalf("unable to initialize k8s clients; %v", err)
	}
	if err := InitWithConfig(k8s.KubeConfig()); err != nil {
		klog.Fatalf("unable to initialize; %v", err)
	}
}

// InitWithConfig initializes the clients with k8s config provided
func InitWithConfig(c *rest.Config) error {
	cs, err := clientset.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("unable to create new clientset interface; %v", err)
	}

	if err := k8s.Init(); err != nil {
		return err
	}

	clientsetInterface = types.NewExtClientset(cs)

	restClient = clientsetInterface.DirectpvLatest().RESTClient()

	if driveClient, err = latestDriveClientForConfig(k8s.KubeConfig()); err != nil {
		return fmt.Errorf("unable to create new drive interface; %v", err)
	}

	if volumeClient, err = latestVolumeClientForConfig(k8s.KubeConfig()); err != nil {
		return fmt.Errorf("unable to create new volume interface; %v", err)
	}

	if nodeClient, err = latestNodeClientForConfig(k8s.KubeConfig()); err != nil {
		return fmt.Errorf("unable to create new node interface; %v", err)
	}

	if initRequestClient, err = latestInitRequestClientForConfig(k8s.KubeConfig()); err != nil {
		return fmt.Errorf("unable to create new initrequest interface; %v", err)
	}

	initEvent(k8s.KubeClient())
	return nil
}
