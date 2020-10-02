// This file is part of MinIO Kubernetes Cloud
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

package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	central "github.com/minio/direct-csi/pkg/controller/central_controller"
	csi "github.com/minio/direct-csi/pkg/controller/csi_controller"
	"github.com/minio/direct-csi/pkg/controller"
	"github.com/minio/direct-csi/pkg/node"
	"github.com/minio/direct-csi/pkg/volume"

	csicommon "github.com/kubernetes-csi/drivers/pkg/csi-common"
	id "github.com/minio/direct-csi/pkg/identity"

	"github.com/golang/glog"
	"github.com/minio/minio/pkg/ellipses"
	"github.com/minio/minio/pkg/mountinfo"
)

// VERSION holds Direct CSI version
const VERSION = "DEVELOPMENT"

// flags
var (
	identity   = "direct.csi.min.io"
	nodeID     = ""
	fsType     = "xfs"
	rack       = "default"
	zone       = "default"
	region     = "default"
	endpoint   = "unix://csi/csi.sock"
	leaderLock = ""
	kubeConfig = ""

	ctx context.Context
)

func init() {
	viper.AutomaticEnv()

	directCSICmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	flag.Set("logtostderr", "true")

	strFlag := func(c *cobra.Command, ptr *string, name string, short string, dfault string, desc string) {
		c.PersistentFlags().
			StringVarP(ptr, name, short, dfault, desc)
	}
	strFlag(directCSICmd, &identity, "identity", "i", identity, "unique name for this CSI driver")
	strFlag(directCSICmd, &kubeConfig, "kube-config", "", kubeConfig, "path to kubeconfig file")

	strFlag(directCSIDriverCmd, &endpoint, "endpoint", "e", endpoint, "endpoint at which direct-csi is listening")
	strFlag(directCSIDriverCmd, &nodeID, "node-id", "n", nodeID, "identity of the node in which direct-csi is running")
	strFlag(directCSIDriverCmd, &fsType, "fs-type", "t", fsType, "default filesystem for device partitions")
	strFlag(directCSIDriverCmd, &rack, "rack", "", rack, "identity of the rack in which this direct-csi is running")
	strFlag(directCSIDriverCmd, &zone, "zone", "", zone, "identity of the zone in which this direct-csi is running")
	strFlag(directCSIDriverCmd, &region, "region", "", region, "identity of the region in which this direct-csi is running")

	strFlag(directCSIControllerCmd, &leaderLock, "leader-lock", "l", identity, "name of the lock used for leader election (defaults to identity)")

	hideFlag := func(name string) {
		directCSICmd.PersistentFlags().MarkHidden(name)
	}
	hideFlag("alsologtostderr")
	hideFlag("log_backtrace_at")
	hideFlag("log_dir")
	hideFlag("logtostderr")
	hideFlag("master")
	hideFlag("stderrthreshold")
	hideFlag("vmodule")

	// suppress the incorrect prefix in glog output
	flag.CommandLine.Parse([]string{})
	viper.BindPFlags(directCSICmd.PersistentFlags())

	var cancel context.CancelFunc

	ctx, cancel = context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)

	go func() {
		s := <-sigs
		cancel()
		panic(fmt.Sprintf("%s %s", s.String(), "Signal received. Exiting"))
	}()

	directCSICmd.AddCommand(directCSICentralControllerCmd, directCSIControllerCmd, directCSIDriverCmd)
}

var directCSICmd = &cobra.Command{
	Use:           "direct-csi",
	Short:         "CSI driver for dynamically provisioning local volumes",
	Long:          "",
	SilenceErrors: true,
	Version:       VERSION,
}

var directCSICentralControllerCmd = &cobra.Command{
	Use:           "central-controller",
	Short:         "run the central-controller for managing resources related to directCSI driver",
	Long:          "",
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return centralControllerManager(args)
	},
}

var directCSIControllerCmd = &cobra.Command{
	Use:           "controller",
	Short:         "run the controller for managing resources related to directCSI driver",
	Long:          "",
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return controllerManager(args)
	},
}

var directCSIDriverCmd = &cobra.Command{
	Use:           "driver",
	Short:         "run the driver for managing resources related to directCSI driver",
	Long:          "",
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return driverManager(args)
	},
}

func driverManager(args []string) error {
	idServer, err := id.NewIdentityServer(identity, VERSION, map[string]string{})
	if err != nil {
		return err
	}

	var basePaths []string
	for _, a := range args {
		if ellipses.HasEllipses(a) {
			p, err := ellipses.FindEllipsesPatterns(a)
			if err != nil {
				return err
			}
			patterns := p.Expand()
			for _, paths := range patterns {
				basePaths = append(basePaths, strings.Join(paths, ""))
			}
		} else {
			basePaths = append(basePaths, a)
		}
	}

	fmter := newFormatMounter()
	mountPaths := make([]string, len(basePaths))
	for i, path := range basePaths {
		blkDev, err := fmter.PathIsDevice(path)
		if err != nil {
			// block device/path not accessible return error
			return err
		}
		mountPath := path
		if blkDev {
			mountPath = fmt.Sprintf("/mnt/drive%d", i) // Using this pattern automatically.
			if err = fmter.MakeDir(mountPath); err != nil {
				return err
			}
			// this is a block device format it and mount.
			// TODO: honor mount options
			if err = fmter.FormatAndMount(path, mountPath, fsType, []string{"rw", "noatime"}); err != nil {
				return err
			}
		}
		mountPaths[i] = mountPath

	}

	// Check for cross device mounts.
	if err = mountinfo.CheckCrossDevice(mountPaths); err != nil {
		return err
	}

	// TODO: support devices
	drives := make([]volume.DriveInfo, len(mountPaths))
	for i, mountPath := range mountPaths {
		drives[i], err = fmter.GetDiskInfo(mountPath)
		if err != nil {
			// fail if one of the drive is not accessible
			return err
		}
	}

	glog.V(10).Infof("base paths: %s", strings.Join(mountPaths, ","))
	volume.InitializeDrives(drives)
	if err := volume.InitializeClient(identity); err != nil {
		return err
	}

	nodeServer, err := node.NewNodeServer(identity, nodeID, rack, zone, region, basePaths)
	if err != nil {
		return err
	}
	glog.V(5).Infof("node server started")

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(endpoint, idServer, nil, nodeServer)
	s.Wait()

	return nil
}

func centralControllerManager(args []string) error {
	c, err := controller.NewDefaultDirectCSIController(identity, leaderLock, 32)
	if err != nil {
		return err
	}
	c.AddStorageTopologyListener(&central.StorageTopologyHandler{
		Identity: identity,
	})

	return c.Run(ctx)
}

func controllerManager(args []string) error {
	idServer, err := id.NewIdentityServer(identity, VERSION, map[string]string{})
	if err != nil {
		return err
	}

	ctrlServer, err := csi.NewControllerServer(identity, nodeID, rack, zone, region)
	if err != nil {
		return err
	}

	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(endpoint, idServer, ctrlServer, nil)
	s.Wait()

	return nil
}

func Run() error {
	return directCSICmd.Execute()
}
