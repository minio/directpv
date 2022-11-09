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
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

const adminServerEnvName = consts.AppCapsName + "_ADMIN_SERVER"

var (
	adminServer string
	forceFlag   bool

	errFormatDenied = errors.New("format denied")
	errFormatFailed = errors.New("format failed")
)

var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Format and add drives",
	Example: strings.ReplaceAll(
		`# Format drives
$ kubectl {PLUGIN_NAME} format

# Format drives from a node
$ kubectl {PLUGIN_NAME} format --node=node1

# Format a drive from all nodes
$ kubectl {PLUGIN_NAME} format --drive-name=nvme1n1

# Format specific drives from specific nodes
$ kubectl {PLUGIN_NAME} format --node=node{1...4} --drive-name=sd{a...f}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if err := validateFormatCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		formatMain(c.Context())
	},
}

func init() {
	addNodeFlag(formatCmd, "If present, select drives from given nodes")
	addDriveNameFlag(formatCmd, "If present, select drives by given names")
	formatCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "If present, force format selected drives")
	formatCmd.PersistentFlags().StringVar(&adminServer, "admin-server", adminServer, fmt.Sprintf("If present, use this value to connect to admin API server instead of %v environment variable", adminServerEnvName))
}

func validateFormatCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}

	if err := validateDriveNameArgs(); err != nil {
		return err
	}

	if adminServer == "" {
		var found bool
		if adminServer, found = os.LookupEnv(adminServerEnvName); !found {
			return fmt.Errorf("environment variable %v or --admin-server argument must be set", adminServerEnvName)
		}
		if adminServer == "" {
			return fmt.Errorf("valid value must be set to %v environment variable", adminServerEnvName)
		}
	}

	host, port, err := net.SplitHostPort(adminServer)
	if err != nil {
		return fmt.Errorf("invalid api server value %v; %w", adminServer, err)
	}
	if host == "" {
		return fmt.Errorf("invalid host of api server value %v", adminServer)
	}
	if port == "" {
		return fmt.Errorf("invalid port number of api server value %v", adminServer)
	}

	return nil
}

func listDevices(ctx context.Context, client *admin.Client) (map[string]admin.ListDevicesResult, error) {
	req := admin.ListDevicesRequest{
		Nodes:         nodeArgs,
		Devices:       driveNameArgs,
		FormatAllowed: true,
	}

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
			"SIZE",
			"FILESYSTEM",
			"MAKE",
		},
		[]table.SortBy{
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
			writer.AppendRow(
				[]interface{}{
					node,
					device.Name,
					printableBytes(int64(device.Size)),
					printableString(device.FSType()),
					printableString(device.Make()),
				},
			)
		}
	}

	if writer.Length() > 0 {
		writer.Render()
	}

	if len(errs) != 0 {
		for node, err := range errs {
			utils.Eprintf(quietFlag, true, "%v: %v\n", node, err)
		}

		return nil, errFormatDenied
	}

	if writer.Length() == 0 {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to format"))
		return nil, errFormatDenied
	}

	return resp.Nodes, nil
}

func getSelections() error {
	if len(nodeArgs) == 0 {
		nodes := getInput(color.HiYellowString("Select nodes (comma separated values, ellipses or ALL):\n"))
		if nodes == "" {
			return errors.New("no node selected")
		}
		if nodes == "ALL" {
			nodeArgs = nil
		} else {
			nodeArgs = strings.Split(nodes, ",")
		}
		if err := validateNodeArgs(); err != nil {
			return err
		}
	}

	if len(driveNameArgs) == 0 {
		devices := getInput(color.HiYellowString("Select drives (comma separated values, ellipses or ALL):\n"))
		if devices == "" {
			return errors.New("no drive selected")
		}
		if devices == "ALL" {
			driveNameArgs = nil
		} else {
			driveNameArgs = strings.Split(devices, ",")
		}
		if err := validateDriveNameArgs(); err != nil {
			return err
		}
	}

	return nil
}

func getFormatDevices(resultMap map[string]admin.ListDevicesResult) (map[string][]admin.FormatDevice, error) {
	writer := newTableWriter(
		table.Row{
			"NODE",
			"NAME",
			"SIZE",
			"FILESYSTEM",
			"MAKE",
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

			if len(driveNameArgs) > 0 && !utils.Contains(driveNameArgs, device.Name) {
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
					printableBytes(int64(device.Size)),
					fsType,
					printableString(device.Make()),
				},
			)
		}
	}

	if len(nodeMap) == 0 {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives selected to format"))
		return nil, nil
	}

	if forceRequired && !forceFlag {
		return nil, fmt.Errorf("--force flag must be set to format drives with known filesystem")
	}

	writer.Render()

	confirm := getInput(color.HiRedString("Format may lead to data loss. Type 'Yes' if you really want to do: "))
	if confirm == "Yes" {
		return nodeMap, nil
	}

	utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives selected to format"))
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
			utils.Eprintf(quietFlag, true, "%v: %v\n", node, err)
		}

		return errFormatFailed
	}

	return nil
}

func formatMain(ctx context.Context) {
	client := admin.NewClient(
		&url.URL{
			Scheme: "https",
			Host:   adminServer,
		},
	)

	resultMap, err := listDevices(ctx, client)
	if err != nil {
		if !errors.Is(err, errFormatDenied) {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}

	if err := getSelections(); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	nodeMap, err := getFormatDevices(resultMap)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if len(nodeMap) == 0 {
		return
	}

	err = formatDevices(ctx, client, nodeMap)
	if err != nil {
		if !errors.Is(err, errFormatFailed) {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}
}
