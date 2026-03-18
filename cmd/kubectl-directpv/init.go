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
			eprintf(true, "Please provide the input file. Check `--help` for usage.\n")
			os.Exit(-1)
		default:
			eprintf(true, "Too many input args. Check `--help` for usage.\n")
			os.Exit(-1)
		}

		if !dangerousFlag {
			eprintf(true, "Initializing the drives will permanently erase existing data. Please review carefully before performing this *DANGEROUS* operation and retry this command with --dangerous flag.\n")
			os.Exit(1)
		}

		initMain(c.Context(), args[0])
	},
}

func init() {
	setFlagOpts(initCmd)

	initCmd.PersistentFlags().DurationVar(&initRequestListTimeout, "timeout", initRequestListTimeout, "specify timeout for the initialization process")
	addDangerousFlag(initCmd, "Perform initialization of drives which will permanently erase existing data")
}

func showResults(results []initResult) {
	if len(results) == 0 {
		return
	}
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

func initDevices(ctx context.Context, initRequests []types.InitRequest, requestID string, teaProgram *tea.Program) (results []initResult, err error) {
	totalReqCount := len(initRequests)
	totalTasks := totalReqCount * 2
	var completedTasks int
	initProgressMap := make(map[string]progressLog, totalReqCount)
	for i := range initRequests {
		initReq, err := adminClient.InitRequest().Create(ctx, &initRequests[i], metav1.CreateOptions{TypeMeta: types.NewInitRequestTypeMeta()})
		if err != nil {
			return nil, err
		}
		if teaProgram != nil {
			completedTasks++
			initProgressMap[initReq.Name] = progressLog{
				log: fmt.Sprintf("Processing initialization request '%s' for node '%v'", initReq.Name, initReq.GetNodeID()),
			}
			teaProgram.Send(progressNotification{
				progressLogs: toProgressLogs(initProgressMap),
				percent:      float64(completedTasks) / float64(totalTasks),
			})
		}
	}
	ctx, cancel := context.WithTimeout(ctx, initRequestListTimeout)
	defer cancel()

	eventCh, stop, err := adminClient.NewInitRequestLister().
		RequestIDSelector(directpvtypes.ToLabelValues([]string{requestID})).
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
			case watch.Modified, watch.Added:
				initReq := event.Item
				if initReq.Status.Status != directpvtypes.InitStatusPending {
					results = append(results, initResult{
						requestID: initReq.Name,
						nodeID:    initReq.GetNodeID(),
						devices:   initReq.Status.Results,
						failed:    initReq.Status.Status == directpvtypes.InitStatusError,
					})
					if teaProgram != nil {
						completedTasks++
						initProgressMap[initReq.Name] = progressLog{
							log:  fmt.Sprintf("Processed initialization request '%s' for node '%v'", initReq.Name, initReq.GetNodeID()),
							done: true,
						}
						teaProgram.Send(progressNotification{
							progressLogs: toProgressLogs(initProgressMap),
							percent:      float64(completedTasks) / float64(totalTasks),
						})
					}
				}
				if len(results) >= totalReqCount {
					return
				}
			case watch.Deleted:
				return
			default:
			}
		case <-ctx.Done():
			err = fmt.Errorf("unable to initialize devices; %w", ctx.Err())
			return
		}
	}
}

func initMain(ctx context.Context, inputFile string) {
	initConfig, err := admin.ReadInitConfig(inputFile)
	if err != nil {
		eprintf(true, "unable to read the input file; %v", err.Error())
		os.Exit(1)
	}

	initRequests, requestID := initConfig.ToInitRequestObjects()
	if len(initRequests) == 0 {
		eprintf(false, "%v\n", color.HiYellowString("No drives are available to init"))
		os.Exit(1)
	}
	defer func() {
		labelMap := map[directpvtypes.LabelKey][]directpvtypes.LabelValue{
			directpvtypes.RequestIDLabelKey: directpvtypes.ToLabelValues([]string{requestID}),
		}
		adminClient.InitRequest().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: directpvtypes.ToLabelSelector(labelMap),
		})
	}()
	var teaProgram *tea.Program
	var wg sync.WaitGroup
	if !quietFlag {
		m := newProgressModel(true)
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
	results, err := initDevices(ctx, initRequests, requestID, teaProgram)
	if err != nil && quietFlag {
		eprintf(true, "%v\n", err)
		os.Exit(1)
	}
	if teaProgram != nil {
		teaProgram.Send(progressNotification{
			done: true,
			err:  err,
		})
		wg.Wait()
	}
	showResults(results)
}
