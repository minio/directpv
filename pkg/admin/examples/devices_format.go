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

	"github.com/minio/directpv/pkg/admin"
)

func main() {
	// Note: ACCESS_KEY, SECRET_KEY are dummy values, please replace them with original values.
	// CAUTION: This example may format the drives. Please be careful when executing this
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

	formatInfo := make(map[admin.NodeName][]admin.FormatDevice)
	for nodeName, deviceList := range result.DeviceInfo {
		var devicesToFormat []admin.FormatDevice
		for _, device := range deviceList {
			devicesToFormat = append(devicesToFormat, admin.FormatDevice{
				Name:       device.Name,
				MajorMinor: device.MajorMinor,
				Force:      device.Filesystem != "",
				UDevData:   device.UDevData,
			})
		}
		if len(devicesToFormat) > 0 {
			formatInfo[admin.NodeName(nodeName)] = devicesToFormat
		}
	}

	if len(formatInfo) == 0 {
		log.Fatal("no devices listed for formatting")
	}

	formatResult, err := admClnt.FormatDevices(context.Background(), admin.FormatDevicesRequest{
		FormatInfo: formatInfo,
	})
	if err != nil {
		log.Fatalln(err)
	}

	for nodeName, formatDeviceStatusList := range formatResult.DeviceInfo {
		for _, formatDeviceStatus := range formatDeviceStatusList {
			if formatDeviceStatus.Error != "" {
				fmt.Printf("\n failed to format device: %s from node: %s due to %s. Error: %s, Suggestion: %s",
					formatDeviceStatus.Name,
					nodeName,
					formatDeviceStatus.Message,
					formatDeviceStatus.Error,
					formatDeviceStatus.Suggestion,
				)
			} else {
				fmt.Printf("\n successfully formatted device: %s from node: %s", formatDeviceStatus.Name, nodeName)
			}
		}
	}
}
