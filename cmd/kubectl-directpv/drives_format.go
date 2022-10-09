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
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/manifoldco/promptui"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

const apiServerEnvName = consts.AppCapsName + "_API_SERVER"

var (
	apiServer   = ""
	allowedFlag = false
	deniedFlag  = false
	forceFlag   = false

	errFormatDenied = errors.New("format denied")
	errFormatFailed = errors.New("format failed")
)

var drivesFormatCmd = &cobra.Command{
	Use:   "format",
	Short: "Format and add drives.",
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

# List all drives by specified combination of node and drive ellipsis for the selection
$ kubectl {PLUGIN_NAME} drives format --drive /dev/xvd{a...d} --node node{1...4}

# Also display unavailable devices in listing
$ kubectl {PLUGIN_NAME} drives format --display-unavailable`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		var err error
		if nodeArgs, err = expandNodeArgs(); err != nil {
			eprintf(fmt.Sprintf("invalid node arguments; %v", err), true)
			os.Exit(-1)
		}
		if driveArgs, err = expandDriveArgs(); err != nil {
			eprintf(fmt.Sprintf("invalid drive arguments; %v", err), true)
			os.Exit(-1)
		}

		if apiServer == "" {
			var found bool
			if apiServer, found = os.LookupEnv(apiServerEnvName); !found {
				eprintf(fmt.Sprintf("environment variable %v or --api-server argument must be set", apiServerEnvName), true)
				os.Exit(-1)
			}
			if apiServer == "" {
				eprintf(fmt.Sprintf("valid value must be set to %v environment variable", apiServerEnvName), true)
				os.Exit(-1)
			}
		}

		host, port, err := net.SplitHostPort(apiServer)
		if err != nil {
			eprintf(fmt.Sprintf("invalid api server; %v", err), true)
			os.Exit(-1)
		}
		if host == "" {
			eprintf("invalid host of api server", true)
			os.Exit(-1)
		}
		if port == "" {
			eprintf("invalid port number of api server", true)
			os.Exit(-1)
		}

		drivesFormatMain(c.Context())
	},
}

func init() {
	drivesFormatCmd.PersistentFlags().BoolVarP(&allowedFlag, "allowed", "", allowedFlag, "Filter output by drives are allowed to format.")
	drivesFormatCmd.PersistentFlags().BoolVarP(&deniedFlag, "denied", "", deniedFlag, "Filter output by drives are denied to format.")
	drivesFormatCmd.PersistentFlags().BoolVarP(&forceFlag, "force", "", forceFlag, "Force format selected drives.")
	drivesFormatCmd.PersistentFlags().StringVarP(&apiServer, "api-server", "", apiServer, "Admin API server in host:port format.")
}

func listDevices(ctx context.Context, client *admin.Client) (map[string]admin.ListDevicesResult, error) {
	req := admin.ListDevicesRequest{
		Nodes:   nodeArgs,
		Devices: driveArgs,
	}
	req.FormatAllowed = allowedFlag
	req.FormatDenied = deniedFlag

	cred, err := admin.GetCredential(ctx, getCredFile())
	if err != nil {
		return nil, err
	}
	resp, err := client.ListDevices(&req, cred)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	writer := newTableWriter(
		table.Row{
			"NODE",
			"NAME",
			"MAKE",
			"SIZE",
			"FILESYSTEM",
			"FORMAT",
			"DENIED",
		},
		[]table.SortBy{
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "FORMAT",
				Mode: table.Asc,
			},
			{
				Name: "NAME",
				Mode: table.Asc,
			},
		},
		false,
	)

	formatDenied := true
	errs := map[string]string{}
	for node, result := range resp.Nodes {
		if result.Error != "" {
			errs[node] = result.Error
			continue
		}

		for _, device := range result.Devices {
			var allow string
			if !device.FormatDenied {
				formatDenied = false
				allow = "Yes"
			}
			writer.AppendRow(
				[]interface{}{
					node,
					device.Name,
					printableString(device.Make()),
					printableBytes(int64(device.Size)),
					printableString(device.FSType()),
					printableString(allow),
					device.DeniedReason,
				},
			)
		}
	}

	if writer.Length() > 0 {
		writer.Render()
	}

	if len(errs) != 0 {
		for node, err := range errs {
			eprintf(fmt.Sprintf("%v: %v", node, err), true)
		}

		return nil, errFormatDenied
	}

	if writer.Length() == 0 || formatDenied {
		fmt.Fprintf(os.Stderr, "%v\n", color.YellowString("No drives found to format"))
		return nil, errFormatDenied
	}

	return resp.Nodes, nil
}

func getInput(msg string) string {
	prompt := promptui.Prompt{
		Label:    msg,
		Validate: func(input string) error { return nil },
	}
	result, err := prompt.Run()
	if err == promptui.ErrInterrupt {
		fmt.Fprintf(os.Stderr, "Exiting by interrupt\n")
		os.Exit(-1)
	}
	return result
}

func getSelections() error {
	if len(nodeArgs) == 0 {
		nodes := getInput(color.YellowString("Select nodes (comma separated values, ellipses or ALL)"))
		if nodes == "" {
			return errors.New("no node selected")
		}
		if nodes == "ALL" {
			nodeArgs = nil
		} else {
			nodeArgs = strings.Split(nodes, ",")
		}
	}

	if len(driveArgs) == 0 {
		devices := getInput(color.YellowString("Select drives (comma separated values, ellipses or ALL)"))
		if devices == "" {
			return errors.New("no drive selected")
		}
		if devices == "ALL" {
			driveArgs = nil
		} else {
			driveArgs = strings.Split(devices, ",")
		}
	}

	var err error
	if nodeArgs, err = expandNodeArgs(); err != nil {
		return fmt.Errorf("invalid node selections; %w", err)
	}
	if driveArgs, err = expandDriveArgs(); err != nil {
		return fmt.Errorf("invalid drive selections; %w", err)
	}

	return nil
}

func getFormatDevices(resultMap map[string]admin.ListDevicesResult) (map[string][]admin.FormatDevice, error) {
	writer := newTableWriter(
		table.Row{
			"NODE",
			"NAME",
			"MAKE",
			"SIZE",
			"FILESYSTEM",
		},
		[]table.SortBy{
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "FORMAT",
				Mode: table.Asc,
			},
			{
				Name: "NAME",
				Mode: table.Asc,
			},
		},
		false,
	)

	forceRequired := false

	nodeMap := map[string][]admin.FormatDevice{}
	for node, result := range resultMap {
		if result.Error != "" {
			continue
		}

		if len(nodeArgs) > 0 && !utils.Contains(nodeArgs, node) {
			continue
		}

		for _, device := range result.Devices {
			if device.FormatDenied {
				continue
			}

			if len(driveArgs) > 0 && !utils.Contains(driveArgs, device.Name) {
				continue
			}

			nodeMap[node] = append(nodeMap[node], admin.NewFormatDevice(device, forceFlag))

			if !forceRequired {
				forceRequired = device.FSType() != ""
			}

			fsType := device.FSType()
			if fsType == "" {
				fsType = "Unknown"
			}

			writer.AppendRow(
				[]interface{}{
					node,
					device.Name,
					printableString(device.Make()),
					printableBytes(int64(device.Size)),
					fsType,
				},
			)
		}
	}

	if len(nodeMap) == 0 {
		return nil, nil
	}

	if forceRequired && !forceFlag {
		return nil, fmt.Errorf("--force flag must be set to format drives with known filesystem")
	}

	writer.Render()

	confirm := getInput(color.HiRedString("Format may lead to data loss. Type 'Yes' if you really want to do"))
	if confirm == "Yes" {
		return nodeMap, nil
	}

	return nil, nil
}

func formatDevices(ctx context.Context, client *admin.Client, nodes map[string][]admin.FormatDevice) error {
	cred, err := admin.GetCredential(ctx, getCredFile())
	if err != nil {
		return err
	}

	req := admin.FormatDevicesRequest{Nodes: nodes}
	resp, err := client.FormatDevices(&req, cred)
	if err != nil {
		return err
	}

	if resp.Error != "" {
		return errors.New(resp.Error)
	}

	writer := newTableWriter(
		table.Row{
			"NODE",
			"NAME",
			"MESSAGE",
		},
		[]table.SortBy{
			{
				Name: "MESSAGE",
				Mode: table.Asc,
			},
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "NAME",
				Mode: table.Asc,
			},
		},
		false,
	)

	errs := map[string]string{}
	for node, result := range resp.Nodes {
		if result.Error != "" {
			errs[node] = result.Error
			continue
		}

		for _, device := range result.Devices {
			msg := "Success"
			if device.Error != "" {
				msg = "ERROR: " + device.Error
			}

			writer.AppendRow(
				[]interface{}{
					node,
					device.Name,
					msg,
				},
			)
		}
	}

	writer.Render()

	if len(errs) != 0 {
		for node, err := range errs {
			eprintf(fmt.Sprintf("%v: %v", node, err), true)
		}

		return errFormatFailed
	}

	return nil
}

func drivesFormatMain(ctx context.Context) {
	client := admin.NewClient(
		&url.URL{
			Scheme: "https",
			Host:   apiServer,
		},
	)

	resultMap, err := listDevices(ctx, client)
	if err != nil {
		if !errors.Is(err, errFormatDenied) {
			eprintf(err.Error(), true)
		}
		os.Exit(1)
	}

	if deniedFlag && !allowedFlag {
		return
	}

	if err := getSelections(); err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}

	nodeMap, err := getFormatDevices(resultMap)
	if err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}

	if len(nodeMap) == 0 {
		return
	}

	err = formatDevices(ctx, client, nodeMap)
	if err != nil {
		if !errors.Is(err, errFormatFailed) {
			eprintf(err.Error(), true)
		}
		os.Exit(1)
	}
}
