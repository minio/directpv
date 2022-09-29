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
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/credential"
	"github.com/minio/directpv/pkg/rest"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	errServerEndpointEnvNotSet = errors.New(consts.APIServerEnv + " not set")
	errInvalidURL              = errors.New("invalid url provided")
	errNoDevicesToFormat       = errors.New("no available devices selected for formatting")
)

var displayUnavailable, debug bool
var formatDrivesCmd = &cobra.Command{
	Use:   "format",
	Short: "Interactively format the devices present in the " + consts.AppPrettyName + " cluster.",
	Example: strings.ReplaceAll(
		`# List all the drives for selection
$ kubectl {PLUGIN_NAME} drives format

# List all drives from a particular node for the selection
$ kubectl {PLUGIN_NAME} drives format --node=node1

# List specified drives from specified nodes for the selection
$ kubectl {PLUGIN_NAME} drives format --node=node1,node2 --drive=/dev/nvme0n1

# List all drives filtered by specified drive ellipsis for the selection
$ kubectl {PLUGIN_NAME} drives format --drive=/dev/sd{a...b}

# List all drives filtered by specified node ellipsis for the selection
$ kubectl {PLUGIN_NAME} drives format --node=node{0...3}

# List all drives by specified combination of node and drive ellipsis
$ kubectl {PLUGIN_NAME} drives format --drive /dev/xvd{a...d} --node node{1...4}

# Also display unavailable devices in listing
$ kubectl {PLUGIN_NAME} drives format --display-unavailable`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	RunE: func(c *cobra.Command, args []string) error {
		// Validate the API Server URL
		var secure bool
		endpointURL, ok := os.LookupEnv(consts.APIServerEnv)
		if !ok {
			return errServerEndpointEnvNotSet
		}
		u, err := url.Parse(endpointURL)
		if err != nil {
			return fmt.Errorf("unable to parse the url %s: %v", endpointURL, err)
		}
		switch u.Scheme {
		case "http":
			secure = false
		case "https":
			secure = true
		default:
			klog.Error("the server url is expected to be in [scheme:][//[host:port]] format")
			return errInvalidURL
		}
		// Load the access and secret keys
		cred, err := credential.Load(configFile)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("credentials not found: please configure access and secret keys via ENV or config file (%s)", configFile)
			}
			return fmt.Errorf("unable to load the credentials: %v", err)
		}
		// initialize rest client
		admClnt, err := rest.New(u.Host, cred.AccessKey, cred.SecretKey, secure)
		if err != nil {
			return fmt.Errorf("unable to initialize the admin client: %v", err)
		}
		if debug {
			admClnt.TraceOn(nil)
		}
		// Expand the args
		if err := expandFormatArgs(); err != nil {
			return err
		}
		return formatDrives(c.Context(), admClnt)
	},
}

func init() {
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&driveArgs, "drive", "d", driveArgs, "Filter by drive path for selection (supports ellipses pattern)")
	formatDrivesCmd.PersistentFlags().StringSliceVarP(&nodeArgs, "node", "n", nodeArgs, "Filter by nodes for selection (supports ellipses pattern)")
	formatDrivesCmd.PersistentFlags().BoolVarP(&displayUnavailable, "display-unavailable", "", displayUnavailable, "Display unavailable devices also during listing")
	formatDrivesCmd.PersistentFlags().BoolVarP(&debug, "debug", "", debug, "Run in debug mode")
}

func formatDrives(ctx context.Context, admClient *rest.AdminClient) (err error) {
	result, err := listDevices(ctx, admClient)
	if err != nil {
		return fmt.Errorf("unable to list devices: %v", err)
	}
	deviceInfo := result.DeviceInfo
	printTable(deviceInfo)

	var rePrint bool
	if len(deviceInfo) > 0 && len(driveArgs) == 0 {
		rePrint = true
		if answer := askQuestion(color.HiBlueString(bold("Please select the drives to format (supports ellipses and comma seperated values)")), nil); answer != "" {
			driveArgs = strings.Split(
				answer,
				",")
		}
	}
	if len(deviceInfo) > 0 && len(nodeArgs) == 0 {
		rePrint = true
		if answer := askQuestion(color.HiBlueString(bold("Please select the nodes (supports ellipses and comma seperated values)")), nil); answer != "" {
			nodeArgs = strings.Split(
				answer,
				",")
		}
	}
	if rePrint {
		if err := expandFormatArgs(); err != nil {
			return err
		}
		deviceInfo, err = filterDevices(ctx, deviceInfo)
		if err != nil {
			return fmt.Errorf("unable to filtert devices: %v", err)
		}
		printTable(deviceInfo)
	}

	if len(deviceInfo) == 0 {
		outputStr := "no devices are available for formatting"
		if len(driveArgs) > 0 || len(nodeArgs) > 0 {
			outputStr = outputStr + ", please try again with correct selectors"
		}
		fmt.Println(color.HiRedString(outputStr + "\n"))
		return nil
	}

	if !ask(color.HiBlueString(fmt.Sprintf(bold("Please confirm if the above listed (available) drives can be formatted (%s|%s)"), answerYes, answerNo))) {
		fmt.Println(color.HiRedString("\nAborting the format request\n"))
		return nil
	}

	formatResult, err := formatDevices(ctx, admClient, deviceInfo)
	if err != nil {
		return err
	}
	for nodeName, formatDeviceStatusList := range formatResult.DeviceInfo {
		for _, formatDeviceStatus := range formatDeviceStatusList {
			if formatDeviceStatus.Error != "" {
				fmt.Println(color.HiRedString("failed to format %s/%s: %s, error: %s, %s",
					nodeName,
					formatDeviceStatus.Name,
					formatDeviceStatus.Message,
					formatDeviceStatus.Error,
					formatDeviceStatus.Suggestion,
				))
			} else {
				fmt.Println(color.HiGreenString("successfully formatted %s/%s", nodeName, formatDeviceStatus.Name))
			}
		}
	}
	return nil
}

func listDevices(ctx context.Context, admClient *rest.AdminClient) (rest.GetDevicesResponse, error) {
	var driveSelectors, nodeSelectors []rest.Selector
	for _, driveArg := range driveArgs {
		driveSelectors = append(driveSelectors, rest.Selector(driveArg))
	}
	for _, nodeArg := range nodeArgs {
		nodeSelectors = append(nodeSelectors, rest.Selector(nodeArg))
	}
	statusSelectors := []rest.DeviceStatus{rest.DeviceStatusAvailable}
	if displayUnavailable {
		statusSelectors = append(statusSelectors, rest.DeviceStatusUnavailable)
	}
	return admClient.ListDevices(context.Background(), rest.GetDevicesRequest{
		Drives:   driveSelectors,
		Nodes:    nodeSelectors,
		Statuses: statusSelectors,
	})
}

func filterDevices(ctx context.Context, deviceInfo map[rest.NodeName][]rest.Device) (map[rest.NodeName][]rest.Device, error) {
	devices := make(map[rest.NodeName][]rest.Device)
	for nodeName, deviceList := range deviceInfo {
		if len(nodeArgs) > 0 && !utils.ItemIn(nodeArgs, string(nodeName)) {
			continue
		}
		for _, device := range deviceList {
			if device.Status == rest.DeviceStatusUnavailable {
				continue
			}
			if len(driveArgs) > 0 && !utils.ItemIn(driveArgs, device.Name) {
				continue
			}
			devices[nodeName] = append(devices[nodeName], device)
		}
	}
	return devices, nil
}

func formatDevices(ctx context.Context, admClient *rest.AdminClient, deviceInfo map[rest.NodeName][]rest.Device) (rest.FormatDevicesResponse, error) {
	formatInfo := make(map[rest.NodeName][]rest.FormatDevice)
	for nodeName, deviceList := range deviceInfo {
		var devicesToFormat []rest.FormatDevice
		for _, device := range deviceList {
			if device.Status == rest.DeviceStatusUnavailable {
				klog.V(5).Infof(color.HiYellowString(italic("skipping %s as it is unavailable", device.Name)))
				continue
			}
			devicesToFormat = append(devicesToFormat, rest.FormatDevice{
				Name:       device.Name,
				MajorMinor: device.MajorMinor,
				Force:      device.Filesystem != "",
				UDevData:   device.UDevData,
			})
		}
		if len(devicesToFormat) > 0 {
			formatInfo[rest.NodeName(nodeName)] = devicesToFormat
		}
	}
	if len(formatInfo) == 0 {
		return rest.FormatDevicesResponse{}, errNoDevicesToFormat
	}
	return admClient.FormatDevices(context.Background(), rest.FormatDevicesRequest{
		FormatInfo: formatInfo,
	})
}
