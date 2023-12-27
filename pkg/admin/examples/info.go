//
//go:build ignore
// +build ignore

// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dustin/go-humanize"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/client"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MaxThreadCount = 200
)

func getKubeConfig() (*rest.Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeConfig := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, nil
}

func main() {
	kubeConfig, err := getKubeConfig()
	if err != nil {
		log.Fatalf("unable to get kubeconfig; %v", err)
	}

	if err := client.InitWithConfig(kubeConfig); err != nil {
		log.Fatalf("unable to initialize client; %v", err)
	}

	nodeInfoMap, err := admin.Info(context.Background())
	if err != nil {
		log.Fatalf("unable to get info; %v", err)
	}

	var totalDriveSize uint64
	var totalVolumeSize uint64
	var totalDriveCount int
	var totalVolumeCount int
	for _, info := range nodeInfoMap {
		totalDriveSize += info.DriveSize
		totalVolumeSize += info.VolumeSize
		totalDriveCount += info.DriveCount
		totalVolumeCount += info.VolumeCount
	}

	fmt.Printf("total volume size: %v\n", humanize.IBytes(totalVolumeSize))
	fmt.Printf("total drive size: %v\n", humanize.IBytes(totalDriveSize))
	fmt.Printf("total volume count: %v\n", totalVolumeCount)
	fmt.Printf("total drive count: %v\n", totalDriveCount)
}
