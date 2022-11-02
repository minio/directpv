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

	"github.com/google/uuid"
	ctrl "github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/fs/xfs"
	id "github.com/minio/directpv/pkg/identity"
	"github.com/minio/directpv/pkg/mount"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils/grpc"
	"github.com/minio/directpv/pkg/volume"
	losetup "gopkg.in/freddierice/go-losetup.v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

var errMountFailure = errors.New("could not mount the drive")

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

	if err = os.Truncate(file.Name(), sys.MinSupportedDeviceSize); err != nil {
		return err
	}

	if err = xfs.MakeFS(ctx, file.Name(), uuid.New().String(), false, reflinkSupport); err != nil {
		klog.V(3).ErrorS(err, "failed to format", "reflink", reflinkSupport)
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

	if err = mount.Mount(loopDevice.Path(), mountPoint, "xfs", []string{mount.MountFlagNoAtime}, mount.MountOptPrjQuota); err != nil {
		klog.V(3).ErrorS(err, "failed to mount", "reflink", reflinkSupport, "flags", mount.MountFlagNoAtime, "mountopts", mount.MountOptPrjQuota)
		return errMountFailure
	}

	return mount.Unmount(mountPoint, true, true, false)
}

func run(ctxMain context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctxMain)
	defer cancel()
	errChan := make(chan error)

	// Start dynamic drive handler container.
	if dynamicDriveHandler {
		return node.RunDynamicDriveHandler(ctx,
			identity,
			nodeID,
			rack,
			zone,
			region,
			disableUDevListener,
		)
	}

	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("identity server started")

	var nodeSrv csi.NodeServer
	if driver {
		var reflinkSupport bool
		// try with reflink enabled
		if err := checkXFS(ctx, true); err == nil {
			reflinkSupport = true
			klog.V(3).Infof("enabled reflink while formatting")
		} else {
			if !errors.Is(err, errMountFailure) {
				return err
			}
			// try with reflink disabled
			if err := checkXFS(ctx, false); err != nil {
				return err
			}
			reflinkSupport = false
			klog.V(3).Infof("disabled reflink while formatting")
		}

		go func() {
			if err := drive.StartController(ctx, nodeID, reflinkSupport); err != nil {
				klog.ErrorS(err, "failed to start drive controller")
				errChan <- err
			}
		}()

		go func() {
			if err := volume.StartController(ctx, nodeID); err != nil {
				klog.ErrorS(err, "failed to start volume controller")
				errChan <- err
			}
		}()

		nodeSrv, err = node.NewNodeServer(ctx,
			identity,
			nodeID,
			rack,
			zone,
			region,
			reflinkSupport,
			metricsPort,
		)
		if err != nil {
			return err
		}
		klog.V(3).Infof("node server started")

	}

	var ctrlServer csi.ControllerServer
	if controller {
		ctrlServer, err = ctrl.NewControllerServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		klog.V(3).Infof("controller manager started")
	}

	go func() {
		if err := grpc.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv); err != nil {
			klog.ErrorS(err, "failed to start grpc server")
			errChan <- err
		}
	}()

	go func() {
		if err := serveReadinessEndpoint(ctx); err != nil {
			klog.ErrorS(err, "failed to serve readiness endpoint")
			errChan <- err
		}
	}()

	err = <-errChan
	if err != nil {
		cancel()
		return err
	}
	return nil
}
