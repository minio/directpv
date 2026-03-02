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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/minio/directpv/pkg/admin/installer"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
)

// Version of this application populated by `go build`
// e.g. $ go build -ldflags="-X main.Version=v4.0.1"
var Version string

// flags
var (
	identity             = consts.Identity
	kubeNodeName         = ""
	rack                 = "default"
	zone                 = "default"
	region               = "default"
	csiEndpoint          = installer.UnixCSIEndpoint
	kubeconfig           = ""
	conversionHealthzURL = ""
	readinessPort        = consts.ReadinessPort

	nodeID directpvtypes.NodeID
)

var mainCmd = &cobra.Command{
	Use:           consts.AppName,
	Short:         "Start " + consts.AppPrettyName + " controller and driver. This binary is usually executed by Kubernetes.",
	SilenceUsage:  true,
	SilenceErrors: false,
	Version:       Version,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd:   true,
		DisableNoDescFlag:   true,
		DisableDescriptions: true,
		HiddenDefaultCmd:    true,
	},
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if kubeNodeName == "" {
			return errors.New("value to --kube-node-name must be provided")
		}

		nodeID = directpvtypes.NodeID(kubeNodeName)

		client.Init()
		return nil
	},
}

func init() {
	if mainCmd.Version == "" {
		mainCmd.Version = "dev"
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
	mainCmd.PersistentFlags().StringVar(&identity, "identity", identity, "Identity of "+consts.AppPrettyName+" instances")
	mainCmd.PersistentFlags().StringVar(&csiEndpoint, "csi-endpoint", csiEndpoint, "CSI endpoint")
	mainCmd.PersistentFlags().StringVar(&kubeNodeName, "kube-node-name", kubeNodeName, "Kubernetes node name (MUST BE SET)")
	mainCmd.PersistentFlags().StringVar(&rack, "rack", rack, "Rack ID of "+consts.AppPrettyName+" instances")
	mainCmd.PersistentFlags().StringVar(&zone, "zone", zone, "Zone ID of "+consts.AppPrettyName+" instances")
	mainCmd.PersistentFlags().StringVar(&region, "region", region, "Region ID of "+consts.AppPrettyName+" instances")
	mainCmd.PersistentFlags().StringVar(&conversionHealthzURL, "conversion-healthz-url", conversionHealthzURL, "URL to conversion webhook health endpoint")
	mainCmd.PersistentFlags().IntVar(&readinessPort, "readiness-port", readinessPort, "Readiness port at "+consts.AppPrettyName+" exports readiness of services")

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

	mainCmd.AddCommand(controllerCmd)
	mainCmd.AddCommand(nodeServerCmd)
	mainCmd.AddCommand(legacyControllerCmd)
	mainCmd.AddCommand(legacyNodeServerCmd)
	mainCmd.AddCommand(nodeControllerCmd)
	mainCmd.AddCommand(repairCmd)
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
		klog.ErrorS(err, "unable to execute command")
		os.Exit(1)
	}
}
