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
	"os"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/clientset"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
)

// Init initializes various clients.
func Init() {
	if atomic.AddInt32(&initialized, 1) != 1 {
		return
	}

	k8s.Init()

	cs, err := clientset.NewForConfig(k8s.KubeConfig())
	if err != nil {
		fmt.Printf("%s: unable to create new clientset interface; %v\n", color.HiRedString("Error"), err)
		os.Exit(1)
	}
	clientsetInterface = types.NewExtClientset(cs)

	restClient = clientsetInterface.DirectpvLatest().RESTClient()

	driveClient, err = latestDriveClientForConfig(k8s.KubeConfig())
	if err != nil {
		fmt.Printf("%s: unable to create new drive interface; %v\n", color.HiRedString("Error"), err)
		os.Exit(1)
	}

	volumeClient, err = latestVolumeClientForConfig(k8s.KubeConfig())
	if err != nil {
		fmt.Printf("%s: unable to create new volume interface; %v\n", color.HiRedString("Error"), err)
		os.Exit(1)
	}

	initEvent(k8s.KubeClient())
}
