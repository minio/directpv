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
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"k8s.io/klog"
)

var Version string

// flags
var (
	kubeconfig     = ""
	identity       = "direct.csi.min.io"
	dryRun         = false
	dryRunFlagName = "dry-run"
)

var pluginCmd = &cobra.Command{
	Use:           "direct-csi",
	Short:         "Plugin for managing Direct CSI drives and volumes",
	SilenceUsage:  true,
	SilenceErrors: false,
	Version:       Version,
}

func init() {
	if pluginCmd.Version == "" {
		pluginCmd.Version = "dev"
	}

	viper.AutomaticEnv()

	flag.Set("alsologtostderr", "true")
	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	// parse the go default flagset to get flags for glog and other packages in future
	pluginCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// defaulting this to true so that logs are printed to console
	flag.Set("logtostderr", "true")

	pluginCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "path to kubeconfig")
	pluginCmd.PersistentFlags().BoolVarP(&dryRun, dryRunFlagName, "", dryRun, "prints the installation yaml")

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

func Execute(ctx context.Context) error {
	return pluginCmd.ExecuteContext(ctx)
}
