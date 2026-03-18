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
	"github.com/minio/directpv/pkg/admin"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/watch"
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
	Run: func(c *cobra.Command, _ []string) {
		if err := validateDiscoverCmd(); err != nil {
			eprintf(true, "%v\n", err)
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
		eprintf(false, "%v\n", color.HiYellowString("No drives are available to initialize"))
		return errDiscoveryFailed
	}

	return nil
}

func writeInitConfig(config admin.InitConfig) error {
	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return config.Write(f)
}

func discoverMain(ctx context.Context) {
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

	nodeCh, errCh, err := adminClient.RefreshNodes(ctx, nodesArgs)
	if err != nil {
		eprintf(true, "discovery failed; %v\n", err)
		os.Exit(1)
	}

	var nodeNames []string
	discoveryProgressMap := make(map[string]progressLog)
	var refreshwg sync.WaitGroup
	refreshwg.Add(1)
	go func() {
		defer refreshwg.Done()
		for {
			select {
			case nodeID, ok := <-nodeCh:
				if !ok {
					return
				}
				nodeNames = append(nodeNames, string(nodeID))
				if teaProgram != nil {
					discoveryProgressMap[string(nodeID)] = progressLog{
						log: fmt.Sprintf("Discovering node '%v'", nodeID),
					}
					teaProgram.Send(progressNotification{
						progressLogs: toProgressLogs(discoveryProgressMap),
					})
				}
			case err, ok := <-errCh:
				if !ok {
					return
				}
				eprintf(true, "discovery failed; %v\n", err)
				os.Exit(1)
			case <-ctx.Done():
				eprintf(true, "%v", ctx.Err().Error())
				os.Exit(1)
			}
		}
	}()
	refreshwg.Wait()

	resultMap, err := discoverDevices(ctx, nodeNames, drivesArgs, teaProgram)
	if err != nil {
		eprintf(true, "discovery failed; %v\n", err)
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
			eprintf(true, "%v\n", err)
		}
		os.Exit(1)
	}
	if err := writeInitConfig(admin.ToInitConfig(resultMap)); err != nil {
		eprintf(true, "unable to write init config; %v\n", err)
	} else if !quietFlag {
		color.HiGreen("Generated '%s' successfully.", outputFile)
	}
}

func discoverDevices(ctx context.Context, nodes, drives []string, teaProgram *tea.Program) (devices map[directpvtypes.NodeID][]types.Device, err error) {
	ctx, cancel := context.WithTimeout(ctx, nodeListTimeout)
	defer cancel()

	eventCh, stop, err := adminClient.NewNodeLister().
		NodeSelector(directpvtypes.ToLabelValues(nodes)).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	discoveryProgressMap := make(map[string]progressLog)
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
				node := event.Item
				if !node.Spec.Refresh {
					devices[directpvtypes.NodeID(node.Name)] = node.GetDevicesByNames(drives)
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
			err = fmt.Errorf("unable to complete the discovery; %w", ctx.Err())
			return
		}
	}
}
