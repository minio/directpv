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
	"fmt"
	"os"
	"path/filepath"

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
	nodeID               = ""
	rack                 = "default"
	zone                 = "default"
	region               = "default"
	endpoint             = "unix://csi/csi.sock"
	kubeconfig           = ""
	controller           = false
	driver               = false
	procfs               = consts.ProcFSDir
	showVersion          = false
	conversionHealthzURL = ""
	metricsPort          = consts.MetricsPort
	readinessPort        = consts.ReadinessPort
)

var driverCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "CSI driver for provisioning from JBOD(s) directly",
	Long: fmt.Sprintf(`This Container Storage Interface (CSI) driver provides just a bunch of drives (JBODs) as volumes consumable within containers. This driver does not manage the lifecycle of the data or the backing of the disk itself. It only acts as the middle-man between a drive and a container runtime.

This driver is rack, region and zone aware i.e., a workload requesting volumes with constraints on rack, region or zone will be scheduled to run only within the constraints. This is useful for requesting volumes that need to be within a specified failure domain (rack, region or zone)

For more information, use '%s --help'
`, os.Args[0]),
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(Version)
			return nil
		}

		switch {
		case controller:
		case driver:
		default:
			return fmt.Errorf("one among [--controller, --driver] should be set")
		}

		client.Init()
		return run(c.Context(), args)
	},
}

func init() {
	if Version == "" {
		Version = "dev"
	}

	viper.AutomaticEnv()

	flag.Set("alsologtostderr", "true")
	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	// parse the go default flagset to get flags for glog and other packages in future
	driverCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	driverCmd.PersistentFlags().AddGoFlagSet(kflags)
	// defaulting this to true so that logs are printed to console
	flag.Set("logtostderr", "true")

	driverCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", kubeconfig, "path to kubeconfig")
	driverCmd.Flags().StringVarP(&identity, "identity", "i", identity, "identity of this "+consts.AppPrettyName)
	driverCmd.Flags().BoolVarP(&showVersion, "version", "", showVersion, "version of "+consts.AppPrettyName)
	driverCmd.Flags().StringVarP(&endpoint, "endpoint", "e", endpoint, "endpoint at which "+consts.AppPrettyName+" is listening")
	driverCmd.Flags().StringVarP(&nodeID, "node-id", "n", nodeID, "identity of the node in which "+consts.AppPrettyName+" is running")
	driverCmd.Flags().StringVarP(&rack, "rack", "", rack, "identity of the rack in which this "+consts.AppPrettyName+" is running")
	driverCmd.Flags().StringVarP(&zone, "zone", "", zone, "identity of the zone in which this "+consts.AppPrettyName+" is running")
	driverCmd.Flags().StringVarP(&region, "region", "", region, "identity of the region in which this "+consts.AppPrettyName+" is running")
	driverCmd.Flags().StringVarP(&procfs, "procfs", "", procfs, "path to host "+consts.ProcFSDir+" for accessing mount information")
	driverCmd.Flags().BoolVarP(&controller, "controller", "", controller, "running in controller mode")
	driverCmd.Flags().BoolVarP(&driver, "driver", "", driver, "run in driver mode")
	driverCmd.Flags().StringVarP(&conversionHealthzURL, "conversion-healthz-url", "", conversionHealthzURL, "The URL of the conversion webhook healthz endpoint")
	driverCmd.Flags().IntVarP(&metricsPort, "metrics-port", "", metricsPort, fmt.Sprintf("Metrics port for scraping. default is %v", consts.MetricsPort))
	driverCmd.Flags().IntVarP(&readinessPort, "readiness-port", "", readinessPort, fmt.Sprintf("Readiness port. default is %v", consts.ReadinessPort))

	driverCmd.PersistentFlags().MarkHidden("alsologtostderr")
	driverCmd.PersistentFlags().MarkHidden("log_backtrace_at")
	driverCmd.PersistentFlags().MarkHidden("log_dir")
	driverCmd.PersistentFlags().MarkHidden("logtostderr")
	driverCmd.PersistentFlags().MarkHidden("master")
	driverCmd.PersistentFlags().MarkHidden("stderrthreshold")
	driverCmd.PersistentFlags().MarkHidden("vmodule")

	// suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(driverCmd.PersistentFlags())
}

// Execute executes driver command.
func Execute(ctx context.Context) error {
	return driverCmd.ExecuteContext(ctx)
}
