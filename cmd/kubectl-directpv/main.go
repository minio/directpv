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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

// Version of this application populated by `go build`
// e.g. $ go build -ldflags="-X main.Version=v4.0.1"
var Version string

// flags
var (
	kubeconfig   = ""
	identity     = consts.Identity
	dryRun       = false
	quiet        = false
	outputFormat = ""
	wideOutput   = false
	jsonOutput   = false
	yamlOutput   = false
	noHeaders    = false
)

var (
	drives, nodes, driveGlobs, nodeGlobs    []string
	driveSelectorValues, nodeSelectorValues []types.LabelValue
	printer                                 func(interface{}) error
)

var mainCmd = &cobra.Command{
	Use:           consts.AppName,
	Short:         "Kubectl plugin for managing " + consts.AppPrettyName + " drives and volumes.",
	SilenceUsage:  true,
	SilenceErrors: false,
	Version:       Version,
	PersistentPreRunE: func(c *cobra.Command, args []string) error {
		switch outputFormat {
		case "":
		case "wide":
			wideOutput = true
		case "yaml":
			yamlOutput = true
		case "json":
			jsonOutput = true
		default:
			return errors.New("'--output' flag value should be one of wide|json|yaml or empty")
		}

		printer = printYAML
		if jsonOutput {
			printer = printJSON
		}

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

	mainCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "Path to the kubeconfig file to use for Kubernetes requests.")
	mainCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", outputFormat, "Output format. One of: json|yaml|wide")
	mainCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "", dryRun, "Run in dry-run mode and output yaml")
	mainCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "", quiet, "Supress printing error logs")
	mainCmd.PersistentFlags().BoolVarP(&noHeaders, "no-headers", "", noHeaders, "When using the default or custom-column output format, don't print headers (default print headers).")

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
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-sigs
		klog.V(1).Infof("Exiting on signal %v; %#v", s.String(), s)
		cancel()
		<-time.After(1 * time.Second)
		os.Exit(1)
	}()

	if err := mainCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, color.HiRedString("ERROR")+" "+err.Error())

		if !quiet {
			fmt.Fprintf(os.Stderr, "run '%s' to get started\n", color.HiWhiteString("kubectl "+consts.AppName+" install"))
		}

		os.Exit(1)
	}
}
