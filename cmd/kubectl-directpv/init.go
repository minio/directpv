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
	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

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

		if !dangerousFlag {
			utils.Eprintf(quietFlag, true, "Initializing the drives will permanently erase existing data. Please review carefully before performing this *DANGEROUS* operation and retry this command with --dangerous flag.\n")
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

func showResults(results []admin.InitResult) {
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
		if result.Failed {
			writer.AppendRow(
				[]interface{}{
					result.RequestID,
					result.NodeID,
					"-",
					color.HiRedString("ERROR; Failed to initialize"),
				},
			)
			continue
		}
		for _, device := range result.Devices {
			msg := "Success"
			if device.Error != "" {
				msg = color.HiRedString("Failed; " + device.Error)
			}
			writer.AppendRow(
				[]interface{}{
					result.RequestID,
					result.NodeID,
					device.Name,
					msg,
				},
			)
		}
	}
	writer.Render()
}

func readInitConfig(inputFile string) (*admin.InitConfig, error) {
	f, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return admin.ParseInitConfig(f)
}

func initMain(ctx context.Context, inputFile string) {
	initConfig, err := readInitConfig(inputFile)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to read the input file; %v", err.Error())
		os.Exit(1)
	}
	results, err := admin.InitDevices(ctx, admin.InitDevicesArgs{
		InitConfig:    initConfig,
		PrintProgress: !quietFlag,
		ListTimeout:   initRequestListTimeout,
	})
	if err != nil {
		if errors.Is(err, admin.ErrNoDrivesAvailableToInit) {
			utils.Eprintf(false, false, "%v\n", color.HiYellowString("No drives are available to init"))
		} else {
			utils.Eprintf(quietFlag, true, "%v\n", err)
		}
		os.Exit(1)
	}
	showResults(results)
}
