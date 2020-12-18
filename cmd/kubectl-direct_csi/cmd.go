// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/golang/glog"
)

var Version string

// flags
var (
	kubeconfig = ""
	identity   = "direct.csi.min.io"
)

var pluginCmd = &cobra.Command{
	Use:           filepath.Base(os.Args[0]),
	Short:         "Plugin for managing Direct CSI drives and volumes",
	Long:          os.Args[0],
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
}

func init() {
	if pluginCmd.Version == "" {
		pluginCmd.Version = "dev"
		Version = "dev"
	}

	viper.AutomaticEnv()
	// parse the go default flagset to get flags for glog and other packages in future
	pluginCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	// defaulting this to true so that logs are printed to console
	flag.Set("logtostderr", "true")

	pluginCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "path to kubeconfig")

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
	// pluginCmd.AddCommand(drivesCmd)
	// pluginCmd.AddCommand(volumesCmd)
}

func Execute(ctx context.Context) error {
	return pluginCmd.ExecuteContext(ctx)
}
