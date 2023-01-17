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
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/table"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type initResult struct {
	requestID string
	nodeID    directpvtypes.NodeID
	failed    bool
	devices   []types.InitDeviceResult
}

var initRequestListTimeout = 2 * time.Minute

var initCmd = &cobra.Command{
	Use:           "init drives.yaml",
	Short:         "Initialize the drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Initialize the drives
   $ kubectl {PLUGIN_NAME} init drives.yaml`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		switch len(args) {
		case 1:
		case 0:
			utils.Eprintf(quietFlag, true, "Please provide the input file. Check `--help` for usage.\n")
			os.Exit(-1)
		default:
			utils.Eprintf(quietFlag, true, "Too many input args. Check `--help` for usage.\n")
			os.Exit(-1)
		}

		input := getInput(color.HiRedString("Initializing the drives will permanently erase existing data. Do you really want to continue (Yes|No)? "))
		if input != "Yes" {
			utils.Eprintf(quietFlag, false, "Aborting...\n")
			os.Exit(1)
		}

		initMain(c.Context(), args[0])
	},
}

func init() {
	initCmd.Flags().SortFlags = false
	initCmd.InheritedFlags().SortFlags = false
	initCmd.LocalFlags().SortFlags = false
	initCmd.LocalNonPersistentFlags().SortFlags = false
	initCmd.NonInheritedFlags().SortFlags = false
	initCmd.PersistentFlags().SortFlags = false

	initCmd.PersistentFlags().DurationVar(&initRequestListTimeout, "timeout", initRequestListTimeout, "specify timeout for the initialization process")
}

func toInitRequestObjects(config *InitConfig, requestID string) (initRequests []types.InitRequest) {
	for _, node := range config.Nodes {
		initDevices := []types.InitDevice{}
		for _, device := range node.Drives {
			if strings.ToLower(device.Select) != driveSelectedValue {
				continue
			}
			initDevices = append(initDevices, types.InitDevice{
				ID:    device.ID,
				Name:  device.Name,
				Force: device.FS != "",
			})
		}
		if len(initDevices) > 0 {
			initRequests = append(initRequests, *types.NewInitRequest(requestID, node.Name, initDevices))
		}
	}
	return
}

func showResults(results []initResult) {
	writer := newTableWriter(
		table.Row{
			"REQUEST_ID",
			"NODE",
			"DRIVE",
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
				Name: "DRIVE",
				Mode: table.Asc,
			},
		},
		false,
	)

	for _, result := range results {
		if result.failed {
			writer.AppendRow(
				[]interface{}{
					result.requestID,
					result.nodeID,
					"-",
					color.HiRedString("ERROR; Failed to initialize"),
				},
			)
			continue
		}
		for _, device := range result.devices {
			msg := "Success"
			if device.Error != "" {
				msg = color.HiRedString("Failed; " + device.Error)
			}
			writer.AppendRow(
				[]interface{}{
					result.requestID,
					result.nodeID,
					device.Name,
					msg,
				},
			)
		}
	}
	writer.Render()
}

func initDevices(ctx context.Context, initRequests []types.InitRequest, requestID string) (results []initResult, err error) {
	var totalReqCount int
	for i := range initRequests {
		_, err := client.InitRequestClient().Create(ctx, &initRequests[i], metav1.CreateOptions{TypeMeta: types.NewInitRequestTypeMeta()})
		if err != nil {
			return nil, err
		}
		totalReqCount++
	}
	ctx, cancel := context.WithTimeout(ctx, initRequestListTimeout)
	defer cancel()

	eventCh, stop, err := initrequest.NewLister().
		RequestIDSelector(toLabelValues([]string{requestID})).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	results = []initResult{}
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
			case watch.Modified:
				initReq := event.InitRequest
				if initReq.Status.Status != directpvtypes.InitStatusPending {
					results = append(results, initResult{
						requestID: initReq.Name,
						nodeID:    initReq.GetNodeID(),
						devices:   initReq.Status.Results,
						failed:    initReq.Status.Status == directpvtypes.InitStatusError,
					})
				}
				if len(results) >= totalReqCount {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			err = fmt.Errorf("unable to initialize devices; %v", ctx.Err())
			return
		}
	}
}

func readInitConfig(inputFile string) (*InitConfig, error) {
	f, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseInitConfig(f)
}

func initMain(ctx context.Context, inputFile string) {
	initConfig, err := readInitConfig(inputFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to read the input file; %v", err.Error())
		os.Exit(1)
	}
	requestID := uuid.New().String()
	initRequests := toInitRequestObjects(initConfig, requestID)
	if len(initRequests) == 0 {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to init"))
		os.Exit(1)
	}
	defer func() {
		labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
			directpvtypes.RequestIDLabelKey: toLabelValues([]string{requestID}),
		}
		client.InitRequestClient().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: directpvtypes.ToLabelSelector(labelMap),
		})
	}()
	results, err := initDevices(ctx, initRequests, requestID)
	if err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
	showResults(results)
}
