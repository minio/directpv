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
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"
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
}

func validateDiscoverCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}
	return validateDriveNameArgs()
}

func toInitConfig(resultMap map[string][]types.Device) InitConfig {
	nodeInfo := []NodeInfo{}
	initConfig := NewInitConfig()
	for node, devices := range resultMap {
		driveInfo := []DriveInfo{}
		for _, device := range devices {
			if device.DeniedReason != "" {
				continue
			}
			driveInfo = append(driveInfo, DriveInfo{
				ID:         device.ID,
				Name:       device.Name,
				MajorMinor: device.MajorMinor,
				FS:         device.FSType,
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

func showDevices(resultMap map[string][]types.Device) error {
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

	var foundAvailableDrive bool
	for node, devices := range resultMap {
		for _, device := range devices {
			var desc string
			available := "YES"
			if device.DeniedReason != "" {
				if !allFlag {
					continue
				}
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
					printableString(device.FSType),
					printableString(device.Make),
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

	if writer.Length() == 0 || !foundAvailableDrive {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to format"))
		return errDiscoveryFailed
	}

	return nil
}

func writeInitConfig(config InitConfig) error {
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return config.Write(f)
}

func discoverDevices(ctx context.Context, nodes []types.Node) (devices map[string][]types.Device, err error) {
	var nodeNames []string
	nodeClient := client.NodeClient()
	for i := range nodes {
		nodeNames = append(nodeNames, nodes[i].Name)
		updateFunc := func() error {
			node, err := nodeClient.Get(ctx, nodes[i].Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			node.Spec.Refresh = true
			if _, err := nodeClient.Update(ctx, node, metav1.UpdateOptions{TypeMeta: types.NewNodeTypeMeta()}); err != nil {
				return err
			}
			return nil
		}
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return nil, err
		}
	}

	nwEventCh, stop, err := node.NewLister().
		NodeSelector(toLabelValues(nodeNames)).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	devices = map[string][]types.Device{}
	for {
		select {
		case nodeEvent, ok := <-nwEventCh:
			if !ok {
				return
			}
			switch nodeEvent.Type {
			case watch.Modified:
				node := nodeEvent.Node
				if !node.Spec.Refresh {
					devices[node.Name] = node.GetDevicesByNames(drivesArgs)
				}
				if len(devices) >= len(nodes) {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

func discoverMain(ctx context.Context) {
	nodes, err := node.NewLister().
		NodeSelector(toLabelValues(nodesArgs)).
		Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if yamlOutput || jsonOutput {
		nodeList := types.NodeList{
			TypeMeta: metav1.TypeMeta{
				Kind:       "List",
				APIVersion: string(directpvtypes.VersionLabelKey),
			},
			Items: nodes,
		}
		if err := printer(nodeList); err != nil {
			utils.Eprintf(quietFlag, true, "unable to %v marshal nodes; %v\n", outputFormat, err)
			os.Exit(1)
		}
		return
	}

	resultMap, err := discoverDevices(ctx, nodes)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}

	if err := showDevices(resultMap); err != nil {
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
