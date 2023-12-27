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
//
// NOTE: The following program may lead do DATA LOSS. Please be careful
// while executing this.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/directpv/pkg/admin"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
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

	if err := admin.Move(context.Background(), admin.MoveArgs{
		Source:      directpvtypes.DriveID("2786de98-2a84-40d4-8cee-8f73686928f8"),
		Destination: directpvtypes.DriveID("b35f1f8e-6bf3-4747-9976-192b23c1a019"),
	}); err != nil {
		log.Fatalf("unable to move the drive; %v", err)
	}
	fmt.Println("successfully moved the drive")
}
