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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/klog"
)

var Version string

// flags
var (
	identity             = "direct-csi-min-io"
	nodeID               = ""
	rack                 = "default"
	zone                 = "default"
	region               = "default"
	endpoint             = "unix://csi/csi.sock"
	kubeconfig           = ""
	controller           = false
	driver               = false
	procfs               = "/proc"
	conversionWebhook    = false
	conversionWebhookURL = ""
	loopBackOnly         = false
	showVersion          = false
)

var driverCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "CSI driver for provisioning from JBOD(s) directly",
	Long: fmt.Sprintf(`
This Container Storage Interface (CSI) driver provides just a bunch of drives (JBODs) as volumes consumable within containers. This driver does not manage the lifecycle of the data or the backing of the disk itself. It only acts as the middle-man between a drive and a container runtime.

This driver is rack, region and zone aware i.e., a workload requesting volumes with constraints on rack, region or zone will be scheduled to run only within the constraints. This is useful for requesting volumes that need to be within a specified failure domain (rack, region or zone)

For more information, use '%s man [sched | examples | ...]'
`, os.Args[0]),
	SilenceUsage: true,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		utils.Init()
	},
	RunE: func(c *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(Version)
			return nil
		}
		if !controller && !driver && !conversionWebhook {
			return fmt.Errorf("one among [--controller, --driver, --conversion-webhook] should be set")
		}
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
	driverCmd.Flags().StringVarP(&identity, "identity", "i", identity, "identity of this direct-csi")
	driverCmd.Flags().BoolVarP(&showVersion, "version", "", showVersion, "version of direct-csi")
	driverCmd.Flags().StringVarP(&endpoint, "endpoint", "e", endpoint, "endpoint at which direct-csi is listening")
	driverCmd.Flags().StringVarP(&nodeID, "node-id", "n", nodeID, "identity of the node in which direct-csi is running")
	driverCmd.Flags().StringVarP(&rack, "rack", "", rack, "identity of the rack in which this direct-csi is running")
	driverCmd.Flags().StringVarP(&zone, "zone", "", zone, "identity of the zone in which this direct-csi is running")
	driverCmd.Flags().StringVarP(&region, "region", "", region, "identity of the region in which this direct-csi is running")
	driverCmd.Flags().StringVarP(&procfs, "procfs", "", procfs, "path to host /proc for accessing mount information")
	driverCmd.Flags().BoolVarP(&controller, "controller", "", controller, "running in controller mode")
	driverCmd.Flags().BoolVarP(&driver, "driver", "", driver, "run in driver mode")
	driverCmd.Flags().BoolVarP(&conversionWebhook, "conversion-webhook", "", conversionWebhook, "start and serve conversion webhook")
	driverCmd.Flags().StringVarP(&conversionWebhookURL, "conversion-webhook-url", "", conversionWebhookURL, "The URL of the conversion webhook")
	driverCmd.Flags().BoolVarP(&loopBackOnly, "loopback-only", "", loopBackOnly, "Create and uses loopback devices only")

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

func Execute(ctx context.Context) error {
	return driverCmd.ExecuteContext(ctx)
}
