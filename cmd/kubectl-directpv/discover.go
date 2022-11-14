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
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	outputFile = "drives.yaml"

	errDiscoveryFailed = errors.New("unable to discover the devices")
)

var discoverCmd = &cobra.Command{
	Use:           "discover",
	Short:         "Discover new drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`# Discover drives
$ kubectl {PLUGIN_NAME} discover

# Discover drives from a node
$ kubectl {PLUGIN_NAME} discover --nodes=node1

# Discover a drive from all nodes
$ kubectl {PLUGIN_NAME} discover --drives=nvme1n1

# Discover specific drives from specific nodes
$ kubectl {PLUGIN_NAME} discover --nodes=node{1...4} --drives=sd{a...f}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if err := validateDiscoverCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}
		discoverMain(c.Context())
	},
}

func init() {
	addNodesFlag(discoverCmd, "If present, select drives from given nodes")
	addDrivesFlag(discoverCmd, "If present, select drives by given names")
	addAllFlag(discoverCmd, "If present, include non-formattable devices in the display")
	discoverCmd.PersistentFlags().StringVar(&outputFile, "output-file", outputFile, "output file to write the init config")
	addAdminServerFlag(discoverCmd)
}

func validateDiscoverCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}
	if err := validateDriveNameArgs(); err != nil {
		return err
	}
	return validateAdminServerConfigArgs()
}

func toInitConfig(resultMap map[string]admin.ListDevicesResult) InitConfig {
	nodeInfo := []NodeInfo{}
	initConfig := NewInitConfig()
	for node, result := range resultMap {
		if result.Error != "" {
			continue
		}
		driveInfo := []DriveInfo{}
		for _, device := range result.Devices {
			if device.FormatDenied {
				continue
			}
			driveInfo = append(driveInfo, DriveInfo{
				ID:         device.ID(types.NodeID(node)),
				Name:       device.Name,
				MajorMinor: device.MajorMinor,
				FS:         device.FSType(),
			})
		}
		nodeInfo = append(nodeInfo, NodeInfo{
			Name:   node,
			Drives: driveInfo,
		})
	}
	initConfig.Nodes = nodeInfo
	return initConfig
}

func listDevices(ctx context.Context, client *admin.Client) (map[string]admin.ListDevicesResult, error) {
	req := admin.ListDevicesRequest{
		Nodes:         nodesArgs,
		Devices:       drivesArgs,
		FormatAllowed: true,
		FormatDenied:  allFlag,
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
			"ID",
			"NODE",
			"DRIVE",
			"SIZE",
			"FILESYSTEM",
			"MAKE",
			"AVAILABLE",
			"DESCRIPTION",
		},
		[]table.SortBy{
			{
				Name: "NODE",
				Mode: table.Asc,
			},
			{
				Name: "DRIVE",
				Mode: table.Asc,
			},
		},
		false,
	)

	errs := map[string]string{}
	var foundAvailableDrive bool
	for node, result := range resp.Nodes {
		if result.Error != "" {
			errs[node] = result.Error
			continue
		}

		for _, device := range result.Devices {
			var desc string
			available := "YES"
			if device.FormatDenied {
				available = "NO"
				desc = device.DeniedReason
			} else {
				foundAvailableDrive = true
			}
			writer.AppendRow(
				[]interface{}{
					device.ID,
					node,
					device.Name,
					printableBytes(int64(device.Size)),
					printableString(device.FSType()),
					printableString(device.Make()),
					available,
					printableString(desc),
				},
			)
		}
	}

	if writer.Length() > 0 {
		writer.Render()
		fmt.Println()
	}

	if len(errs) != 0 {
		for node, err := range errs {
			utils.Eprintf(quietFlag, true, "%v: %v\n", node, err)
		}

		return nil, errDiscoveryFailed
	}

	if writer.Length() == 0 || !foundAvailableDrive {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to format"))
		return nil, errDiscoveryFailed
	}

	return resp.Nodes, nil
}

func writeInitConfig(config InitConfig) error {
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return config.Write(f)
}

func discoverMain(ctx context.Context) {
	client := admin.NewClient(
		&url.URL{
			Scheme: "https",
			Host:   adminServerArg,
		},
	)

	resultMap, err := listDevices(ctx, client)
	if err != nil {
		if !errors.Is(err, errDiscoveryFailed) {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}

	if err := writeInitConfig(toInitConfig(resultMap)); err != nil {
		utils.Eprintf(quietFlag, true, "unable to write init config; %v\n", err)
	} else if !quietFlag {
		color.HiGreen("Generated '%s' successfully.", outputFile)
	}
}
