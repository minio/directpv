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
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	outputFile         = "drives.yaml"
	errDiscoveryFailed = errors.New("unable to discover the devices")
	nodeListTimeout    = 2 * time.Minute
)

var discoverCmd = &cobra.Command{
	Use:           "discover",
	Short:         "Discover new drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Discover drives
   $ kubectl {PLUGIN_NAME} discover

2. Discover drives from a node
   $ kubectl {PLUGIN_NAME} discover --nodes=node1

3. Discover a drive from all nodes
   $ kubectl {PLUGIN_NAME} discover --drives=nvme1n1

4. Discover all drives from all nodes (including unavailable)
   $ kubectl {PLUGIN_NAME} discover --all

5. Discover specific drives from specific nodes
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
	setFlagOpts(discoverCmd)

	addNodesFlag(discoverCmd, "discover drives from given nodes")
	addDrivesFlag(discoverCmd, "discover drives by given names")
	addAllFlag(discoverCmd, "If present, include non-formattable devices in the display")
	discoverCmd.PersistentFlags().StringVar(&outputFile, "output-file", outputFile, "output file to write the init config")
	discoverCmd.PersistentFlags().DurationVar(&nodeListTimeout, "timeout", nodeListTimeout, "specify timeout for the discovery process")
}

func validateDiscoverCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}
	return validateDriveNameArgs()
}

func toInitConfig(resultMap map[directpvtypes.NodeID][]types.Device) InitConfig {
	nodeInfo := []NodeInfo{}
	initConfig := NewInitConfig()
	for node, devices := range resultMap {
		driveInfo := []DriveInfo{}
		for _, device := range devices {
			if device.DeniedReason != "" {
				continue
			}
			driveInfo = append(driveInfo, DriveInfo{
				ID:     device.ID,
				Name:   device.Name,
				Size:   device.Size,
				Make:   device.Make,
				FS:     device.FSType,
				Select: driveSelectedValue,
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

func showDevices(resultMap map[directpvtypes.NodeID][]types.Device) error {
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
					device.ID[:16] + "...",
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
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to initialize"))
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

func discoverDevices(ctx context.Context, nodes []types.Node, teaProgram *tea.Program) (devices map[directpvtypes.NodeID][]types.Device, err error) {
	var nodeNames []string
	nodeClient := client.NodeClient()
	totalNodeCount := len(nodes)
	discoveryProgressMap := make(map[string]progressLog, totalNodeCount)
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
			if teaProgram != nil {
				discoveryProgressMap[node.Name] = progressLog{
					log: fmt.Sprintf("Discovering node '%v'", node.Name),
				}
				teaProgram.Send(progressNotification{
					progressLogs: toProgressLogs(discoveryProgressMap),
				})
			}
			return nil
		}
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, nodeListTimeout)
	defer cancel()

	eventCh, stop, err := node.NewLister().
		NodeSelector(toLabelValues(nodeNames)).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	devices = map[directpvtypes.NodeID][]types.Device{}
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			if event.Err != nil {
				err = event.Err
				return
			}
			switch event.Type {
			case watch.Modified, watch.Added:
				node := event.Node
				if !node.Spec.Refresh {
					devices[directpvtypes.NodeID(node.Name)] = node.GetDevicesByNames(drivesArgs)
					if teaProgram != nil {
						discoveryProgressMap[node.Name] = progressLog{
							log:  fmt.Sprintf("Discovered node '%v'", node.Name),
							done: true,
						}
						teaProgram.Send(progressNotification{
							progressLogs: toProgressLogs(discoveryProgressMap),
						})
					}
				}
				if len(devices) >= len(nodes) {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			utils.Eprintf(quietFlag, true, "unable to complete the discovery; %v\n", ctx.Err())
			return
		}
	}
}

func syncNodes(ctx context.Context) (err error) {
	csiNodes, err := getCSINodes(ctx)
	if err != nil {
		return fmt.Errorf("unable to get CSI nodes; %w", err)
	}

	nodes, err := node.NewLister().Get(ctx)
	if err != nil {
		return fmt.Errorf("unable to get nodes; %w", err)
	}

	var nodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}

	// Add missing nodes.
	for _, csiNode := range csiNodes {
		if !utils.Contains(nodeNames, csiNode) {
			node := types.NewNode(directpvtypes.NodeID(csiNode), nil)
			node.Spec.Refresh = true
			if _, err = client.NodeClient().Create(ctx, node, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("unable to create node %v; %w", csiNode, err)
			}
		}
	}

	// Remove non-existing nodes.
	for _, nodeName := range nodeNames {
		if !utils.Contains(csiNodes, nodeName) {
			if err = client.NodeClient().Delete(ctx, nodeName, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("unable to remove non-existing node %v; %w", nodeName, err)
			}
		}
	}

	return nil
}

func discoverMain(ctx context.Context) {
	if err := syncNodes(ctx); err != nil {
		utils.Eprintf(quietFlag, true, "sync failed; %v\n", err)
		os.Exit(1)
	}

	nodes, err := node.NewLister().
		NodeSelector(toLabelValues(nodesArgs)).
		Get(ctx)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	if len(nodesArgs) != 0 && len(nodes) == 0 {
		suffix := ""
		if len(nodesArgs) > 1 {
			suffix = "s"
		}
		utils.Eprintf(quietFlag, true, "node%v %v not found\n", suffix, nodesArgs)
		os.Exit(1)
	}

	var teaProgram *tea.Program
	var wg sync.WaitGroup
	if !quietFlag {
		m := newProgressModel(false)
		teaProgram = tea.NewProgram(m)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := teaProgram.Run(); err != nil {
				fmt.Println("error running program:", err)
				os.Exit(1)
			}
		}()
	}
	resultMap, err := discoverDevices(ctx, nodes, teaProgram)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	if teaProgram != nil {
		teaProgram.Send(progressNotification{
			done: true,
			err:  err,
		})
		wg.Wait()
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
