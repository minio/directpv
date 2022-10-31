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
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

// Version of this application populated by `go build`
// e.g. $ go build -ldflags="-X main.Version=v4.0.1"
var Version string

var mainCmd = &cobra.Command{
	Use:           consts.AppName,
	Short:         "Kubectl plugin for managing " + consts.AppPrettyName + " drives and volumes.",
	SilenceUsage:  false,
	SilenceErrors: false,
	Version:       Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		client.Init()
		return nil
	},
}

func init() {
	if Version == "" {
		mainCmd.Version = "0.0.0-dev"
		image = consts.AppName + ":0.0.0-dev"
	}

	viper.AutomaticEnv()

	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	// parse the go default flagset to get flags for glog and other packages in future
	mainCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	mainCmd.PersistentFlags().AddGoFlagSet(kflags)

	flag.Set("logtostderr", "true")
	flag.Set("alsologtostderr", "true")

	mainCmd.PersistentFlags().StringVarP(
		&kubeconfig,
		"kubeconfig",
		"",
		kubeconfig,
		"Path to the kubeconfig file to use for CLI requests.",
	)
	mainCmd.PersistentFlags().BoolVarP(
		&quietFlag,
		"quiet",
		"",
		quietFlag,
		"Suppress printing error messages",
	)

	mainCmd.PersistentFlags().MarkHidden("alsologtostderr")
	mainCmd.PersistentFlags().MarkHidden("add_dir_header")
	mainCmd.PersistentFlags().MarkHidden("log_file")
	mainCmd.PersistentFlags().MarkHidden("log_file_max_size")
	mainCmd.PersistentFlags().MarkHidden("one_output")
	mainCmd.PersistentFlags().MarkHidden("skip_headers")
	mainCmd.PersistentFlags().MarkHidden("skip_log_headers")
	mainCmd.PersistentFlags().MarkHidden("v")
	mainCmd.PersistentFlags().MarkHidden("log_backtrace_at")
	mainCmd.PersistentFlags().MarkHidden("log_dir")
	mainCmd.PersistentFlags().MarkHidden("logtostderr")
	mainCmd.PersistentFlags().MarkHidden("master")
	mainCmd.PersistentFlags().MarkHidden("stderrthreshold")
	mainCmd.PersistentFlags().MarkHidden("vmodule")

	// suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(mainCmd.PersistentFlags())

	mainCmd.AddCommand(infoCmd)
	mainCmd.AddCommand(installCmd)
	mainCmd.AddCommand(uninstallCmd)
	mainCmd.AddCommand(getCmd)
	mainCmd.AddCommand(cordonCmd)
	mainCmd.AddCommand(uncordonCmd)
	mainCmd.AddCommand(formatCmd)
	mainCmd.AddCommand(moveCmd)
	mainCmd.AddCommand(releaseCmd)
	mainCmd.AddCommand(setCmd)
	mainCmd.AddCommand(purgeCmd)
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())

	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case signal := <-signalCh:
			eprintf(quietFlag, false, "\nExiting on signal %v\n", signal)
			cancelFunc()
			os.Exit(1)
		case <-ctx.Done():
		}
	}()

	if err := mainCmd.ExecuteContext(ctx); err != nil {
		eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
}
