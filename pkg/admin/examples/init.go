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
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/client"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MaxThreadCount = 200
)

var initConfigJson = `{
  "version": "v1",
  "nodes": [
    {
      "name": "praveen-thinkpad-x1-carbon-6th",
      "drives": [
        {
          "id": "253:0$GMWf7RjztKikTRjxdlsl8fo4uxIO3j8s/kHg2H18UA8=",
          "name": "dm-0",
          "size": 2042626048,
          "make": "vg0-lv--0",
          "fs": "xfs",
          "select": "yes"
        },
        {
          "id": "253:1$AiLJgxtVolBaB7U43uSpPlChoGvrRPVPGXrB7Y8T1Uc=",
          "name": "dm-1",
          "size": 2042626048,
          "make": "vg0-lv--1",
          "fs": "xfs",
          "select": "yes"
        },
        {
          "id": "253:3$5bO913qnAU5w64tDdQbNjGWJTmqcgXM8UEO0rMrbsCs=",
          "name": "dm-3",
          "size": 2042626048,
          "make": "vg0-lv--3",
          "fs": "xfs",
          "select": "yes"
        },
        {
          "id": "253:2$hH2+3aySmMa9azQmtCythl3/64X23HdUOCmGoHQBjNE=",
          "name": "dm-2",
          "size": 2042626048,
          "make": "vg0-lv--2",
          "fs": "xfs",
          "select": "yes"
        }
      ]
    }
  ]
}`

var initConfigYaml = `version: v1
nodes:
    - name: praveen-thinkpad-x1-carbon-6th
      drives:
        - id: 253:0$GMWf7RjztKikTRjxdlsl8fo4uxIO3j8s/kHg2H18UA8=
          name: dm-0
          size: 2042626048
          make: vg0-lv--0
          fs: xfs
          select: "yes"
        - id: 253:1$AiLJgxtVolBaB7U43uSpPlChoGvrRPVPGXrB7Y8T1Uc=
          name: dm-1
          size: 2042626048
          make: vg0-lv--1
          fs: xfs
          select: "yes"
        - id: 253:3$5bO913qnAU5w64tDdQbNjGWJTmqcgXM8UEO0rMrbsCs=
          name: dm-3
          size: 2042626048
          make: vg0-lv--3
          fs: xfs
          select: "yes"
        - id: 253:2$hH2+3aySmMa9azQmtCythl3/64X23HdUOCmGoHQBjNE=
          name: dm-2
          size: 2042626048
          make: vg0-lv--2
          fs: xfs
          select: "yes"
`

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

	initConfig, err := admin.ParseInitConfig(strings.NewReader(initConfigJson))
	if err != nil {
		log.Fatalf("unable to parse init config; %v", err)
	}

	results, err := admin.InitDevices(context.Background(), admin.InitDevicesArgs{
		InitConfig: initConfig,
	})
	if err != nil {
		log.Fatalf("unable to initialize the device; %v", err)
	}

	for _, result := range results {
		fmt.Printf("RequestID: %v\n", result.RequestID)
		fmt.Printf("NodeID: %v\n", result.NodeID)
		fmt.Printf("Failed: %v\n", result.Failed)
		var initializedDevices, failedDevices []string
		for _, device := range result.Devices {
			if device.Error == "" {
				initializedDevices = append(initializedDevices, device.Name)
			} else {
				failedDevices = append(failedDevices, device.Name+"("+device.Error+")")
			}
		}
		if len(initializedDevices) > 0 {
			fmt.Printf("Devices initialized: %v\n", strings.Join(initializedDevices, ","))
		}
		if len(failedDevices) > 0 {
			fmt.Printf("Devices failed: %v\n", strings.Join(failedDevices, ","))
		}
		fmt.Println("---")
	}
}
