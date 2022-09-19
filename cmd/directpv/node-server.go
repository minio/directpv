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

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/consts"
	pkgidentity "github.com/minio/directpv/pkg/identity"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/volume"
	"github.com/minio/directpv/pkg/xfs"
	"github.com/spf13/cobra"
	losetup "gopkg.in/freddierice/go-losetup.v1"
	"k8s.io/klog/v2"
)

var errMountFailure = errors.New("could not mount the drive")

var nodeServerCmd = &cobra.Command{
	Use:           "node-server",
	Short:         "Start node server of " + consts.AppPrettyName + ".",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		return startNodeServer(c.Context(), args)
	},
}

func init() {
	nodeServerCmd.PersistentFlags().IntVarP(&metricsPort, "metrics-port", "", metricsPort, "Metrics port at "+consts.AppPrettyName+" exports metrics data")
}

func checkXFS(ctx context.Context, reflinkSupport bool) error {
	mountPoint, err := os.MkdirTemp("", "xfs.check.mnt.")
	if err != nil {
		return err
	}
	defer os.Remove(mountPoint)

	file, err := os.CreateTemp("", "xfs.check.file.")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	file.Close()

	if err = os.Truncate(file.Name(), xfs.MinSupportedDeviceSize); err != nil {
		return err
	}

	if err = xfs.MakeFS(ctx, file.Name(), uuid.New().String(), false, reflinkSupport); err != nil {
		klog.V(3).ErrorS(err, "unable to make XFS filesystem", "reflink", reflinkSupport)
		return err
	}

	loopDevice, err := losetup.Attach(file.Name(), 0, false)
	if err != nil {
		return err
	}

	defer func() {
		if err := loopDevice.Detach(); err != nil {
			klog.Error(err)
		}
	}()

	if err = xfs.Mount(loopDevice.Path(), mountPoint); err != nil {
		klog.V(3).ErrorS(err, "unable to mount XFS filesystem", "reflink", reflinkSupport)
		return errMountFailure
	}

	return sys.Unmount(mountPoint, true, true, false)
}

func getReflinkSupport(ctx context.Context) (reflinkSupport bool, err error) {
	reflinkSupport = true
	if err = checkXFS(ctx, reflinkSupport); err != nil {
		if errors.Is(err, errMountFailure) {
			reflinkSupport = false
			err = checkXFS(ctx, reflinkSupport)
		}
	}
	return
}

func startNodeServer(ctx context.Context, args []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	idServer, err := pkgidentity.NewServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("Identity server started")

	reflinkSupport, err := getReflinkSupport(ctx)
	if err != nil {
		return err
	}

	if reflinkSupport {
		klog.V(3).Infof("reflink support is ENABLED for XFS formatting and mounting")
	} else {
		klog.V(3).Infof("reflink support is DISABLED for XFS formatting and mounting")
	}

	errCh := make(chan error)

	go func() {
		if err := volume.StartController(ctx, kubeNodeName); err != nil {
			klog.ErrorS(err, "unable to start volume controller")
			errCh <- err
		}
	}()

	var nodeServer csi.NodeServer
	nodeServer, err = node.NewServer(
		ctx,
		identity,
		kubeNodeName,
		rack,
		zone,
		region,
		reflinkSupport,
		metricsPort,
	)
	if err != nil {
		return err
	}
	klog.V(3).Infof("Node server started")

	if err = os.Mkdir(consts.MountRootDir, 0o777); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	go func() {
		if err := runServers(ctx, csiEndpoint, idServer, nil, nodeServer); err != nil {
			klog.ErrorS(err, "unable to start GRPC servers")
			errCh <- err
		}
	}()

	go func() {
		if err := serveReadinessEndpoint(ctx); err != nil {
			klog.ErrorS(err, "unable to start readiness endpoint")
			errCh <- err
		}
	}()

	return <-errCh
}
