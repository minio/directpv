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
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
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

var errInitFailed = errors.New("init failed")

const (
	initRequestListTimeout = 2 * time.Minute
)

var initCmd = &cobra.Command{
	Use:           "init drives.yaml",
	Short:         "Initialize the drives",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`# Initialize the drives
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

func toInitRequestObjects(config *InitConfig) (initRequests []types.InitRequest) {
	for _, node := range config.Nodes {
		initDevices := []types.InitDevice{}
		for _, device := range node.Drives {
			initDevices = append(initDevices, types.InitDevice{
				ID:         device.ID,
				Name:       device.Name,
				MajorMinor: device.MajorMinor,
				Force:      device.FS != "",
			})
		}
		if len(initDevices) > 0 {
			initRequests = append(initRequests, *types.NewInitRequest(directpvtypes.NodeID(node.Name), initDevices))
		}
	}
	return
}

func showResults(results map[string][]types.InitDeviceResult) {
	writer := newTableWriter(
		table.Row{
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

	for node, devices := range results {
		for _, device := range devices {
			msg := "Success"
			if device.Error != "" {
				msg = "Failed; " + device.Error
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
}

func initDevices(ctx context.Context, initRequests []types.InitRequest) (results map[string][]types.InitDeviceResult, err error) {
	if len(initRequests) == 0 {
		utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to init"))
		return nil, errInitFailed
	}

	var namesToWatch []string
	for i := range initRequests {
		initR, err := client.InitRequestClient().Create(ctx, &initRequests[i], metav1.CreateOptions{TypeMeta: types.NewInitRequestTypeMeta()})
		if err != nil {
			return nil, err
		}
		namesToWatch = append(namesToWatch, initR.Name)
	}
	ctx, cancel := context.WithTimeout(ctx, initRequestListTimeout)
	defer cancel()

	eventCh, stop, err := initrequest.NewLister().
		InitRequestNameSelector(namesToWatch).
		Watch(ctx)
	if err != nil {
		return nil, err
	}
	defer stop()

	results = map[string][]types.InitDeviceResult{}
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			switch event.Type {
			case watch.Modified:
				initReq := event.InitRequest
				if initReq.Status.Status != directpvtypes.InitStatusPending {
					results[string(initReq.GetNodeID())] = initReq.Status.Results
				}
				if len(results) >= len(namesToWatch) {
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
	results, err := initDevices(ctx, toInitRequestObjects(initConfig))
	if err != nil {
		if !errors.Is(err, errInitFailed) {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}
	showResults(results)
}
