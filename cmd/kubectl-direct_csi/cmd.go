// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"k8s.io/klog/v2"

	"github.com/minio/direct-csi/pkg/utils"
)

// Version is kubectl direct-csi version.
var Version string

// flags
var (
	kubeconfig = ""
	identity   = "direct.csi.min.io"
	dryRun     = false
	//output modes
	outputMode = ""
	wide       = false
	json       = false
	yaml       = false
)

var drives, nodes, driveGlobs, driveSelectors, nodeGlobs, nodeSelectors []string

var printer func(interface{}) error

var pluginCmd = &cobra.Command{
	Use:           "direct-csi",
	Short:         "Plugin for managing Direct CSI drives and volumes",
	SilenceUsage:  true,
	SilenceErrors: false,
	Version:       Version,
	PersistentPreRunE: func(c *cobra.Command, args []string) error {
		utils.Init()

		switch outputMode {
		case "":
		case "wide":
			wide = true
		case "yaml":
			yaml = true
		case "json":
			json = true
		default:
			return errors.New("output should be one of wide|json|yaml or empty")
		}

		printer = printYAML
		if json {
			printer = printJSON
		}

		return nil
	},
}

func init() {
	if pluginCmd.Version == "" {
		pluginCmd.Version = "dev"
	}

	viper.AutomaticEnv()

	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	// parse the go default flagset to get flags for glog and other packages in future
	pluginCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	pluginCmd.PersistentFlags().AddGoFlagSet(kflags)

	flag.Set("logtostderr", "true")
	flag.Set("alsologtostderr", "true")

	pluginCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "path to kubeconfig")
	pluginCmd.PersistentFlags().StringVarP(&outputMode, "output", "o", outputMode,
		"output format should be one of wide|json|yaml or empty")
	pluginCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", dryRun, "prints the installation yaml")

	pluginCmd.PersistentFlags().MarkHidden("alsologtostderr")
	pluginCmd.PersistentFlags().MarkHidden("log_backtrace_at")
	pluginCmd.PersistentFlags().MarkHidden("log_dir")
	pluginCmd.PersistentFlags().MarkHidden("logtostderr")
	pluginCmd.PersistentFlags().MarkHidden("master")
	pluginCmd.PersistentFlags().MarkHidden("stderrthreshold")
	pluginCmd.PersistentFlags().MarkHidden("vmodule")

	// suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(pluginCmd.PersistentFlags())

	pluginCmd.AddCommand(infoCmd)
	pluginCmd.AddCommand(installCmd)
	pluginCmd.AddCommand(uninstallCmd)
	pluginCmd.AddCommand(drivesCmd)
	pluginCmd.AddCommand(volumesCmd)
	//pluginCmd.AddCommand(newVolumesCmd())
}

// Execute executes plugin command.
func Execute(ctx context.Context) error {
	return pluginCmd.ExecuteContext(ctx)
}
