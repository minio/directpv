//go:build ignore
// +build ignore

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

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dustin/go-humanize"
	"github.com/minio/directpv/pkg/admin"
)

func main() {
	// Note: ACCESS_KEY, SECRET_KEY are dummy values, please replace them with original values.
	admClnt, err := admin.New("<your-api-server-host>:40443", "ACCESS_KEY", "SECRET_KEY", true)
	if err != nil {
		log.Fatalln(err)
	}

	// Uncomment to enable trace
	// admClnt.TraceOn(nil)

	result, err := admClnt.ListDevices(context.Background(), admin.GetDevicesRequest{
		Drives:   []admin.Selector{},
		Nodes:    []admin.Selector{},
		Statuses: []admin.DeviceStatus{}, // possible values are admin.DeviceStatusAvailable and admin.DeviceStatusUnavailable
	})
	if err != nil {
		log.Fatalln(err)
	}

	for nodeName, deviceList := range result.DeviceInfo {
		fmt.Printf("\n-------------------- Devices from Node: %s -----------------------------\n", nodeName)
		for _, device := range deviceList {
			fmt.Printf(" Device: %s", device.Name)
			fmt.Printf("\n MajorMinor: %s", device.MajorMinor)
			fmt.Printf("\n Size: %s", humanize.IBytes(device.Size))
			fmt.Printf("\n Model: %s", device.Model)
			fmt.Printf("\n Vendor: %s", device.Vendor)
			fmt.Printf("\n Filesystem: %s", device.Filesystem)
			fmt.Printf("\n Status: %s", device.Status)
			fmt.Println("\n---------XX---------")
		}
	}
}
