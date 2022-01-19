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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
	ctrl "github.com/minio/directpv/pkg/controller"
	"github.com/minio/directpv/pkg/converter"
	"github.com/minio/directpv/pkg/fs/xfs"
	id "github.com/minio/directpv/pkg/identity"
	"github.com/minio/directpv/pkg/node"
	"github.com/minio/directpv/pkg/node/discovery"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils/grpc"
	"github.com/minio/directpv/pkg/volume"
	losetup "gopkg.in/freddierice/go-losetup.v1"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

const (
	conversionCAFile = "/etc/conversion/CAs/ca.pem"
)

var (
	errInvalidConversionHealthzURL = errors.New("the `--conversion-webhook-healthz-url` flag is unset/empty")
)

func waitForConversionWebhook() error {
	if conversionHealthzURL == "" {
		return errInvalidConversionHealthzURL
	}

	caCert, err := os.ReadFile(conversionCAFile)
	if err != nil {
		klog.V(2).Infof("Error while reading cacert %v", err)
		return err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
		Timeout: 2 * time.Second,
	}
	defer client.CloseIdleConnections()

	if err := retry.OnError(
		wait.Backoff{
			Duration: 1 * time.Second,
			Factor:   2.0,
			Jitter:   0.1,
			Steps:    5,
			Cap:      1 * time.Minute,
		},
		func(err error) bool {
			return err != nil
		},
		func() error {
			_, err := client.Get(conversionHealthzURL)
			if err != nil {
				klog.V(2).Infof("Waiting for conversion webhook: %v", err)
			}
			return err
		}); err != nil {
		return err
	}

	return nil
}

func checkXFS(ctx context.Context) (bool, error) {
	mountPoint, err := os.MkdirTemp("", "xfs.check.mnt.")
	if err != nil {
		return false, err
	}
	defer os.Remove(mountPoint)

	file, err := os.CreateTemp("", "xfs.check.file.")
	if err != nil {
		return false, err
	}
	defer os.Remove(file.Name())
	file.Close()

	if err = os.Truncate(file.Name(), sys.MinSupportedDeviceSize); err != nil {
		return false, err
	}

	if err = xfs.MakeFS(ctx, file.Name(), uuid.New().String(), false, true); err != nil {
		return false, err
	}

	loopDevice, err := losetup.Attach(file.Name(), 0, false)
	if err != nil {
		return false, err
	}

	defer func() {
		if err := loopDevice.Detach(); err != nil {
			klog.Error(err)
		}
	}()

	if err = sys.Mount(loopDevice.Path(), mountPoint, "xfs", nil, ""); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			err = nil
		}
		return false, err
	}

	return true, sys.Unmount(mountPoint, true, true, false)
}

func run(ctx context.Context, args []string) error {

	// Start conversion webserver
	if err := converter.ServeConversionWebhook(ctx); err != nil {
		return err
	}

	if err := waitForConversionWebhook(); err != nil {
		return err
	}
	klog.V(3).Info("The conversion webhook is live!")

	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	klog.V(3).Infof("identity server started")

	var nodeSrv csi.NodeServer
	if driver {
		reflinkSupport, err := checkXFS(ctx)
		if err != nil {
			return err
		}

		if !dynamicDriveDiscovery {
			discovery, err := discovery.NewDiscovery(ctx, identity, nodeID, rack, zone, region)
			if err != nil {
				return err
			}
			if err := discovery.Init(ctx, loopbackOnly); err != nil {
				return fmt.Errorf("error while initializing drive discovery: %v", err)
			}
			klog.V(3).Infof("Drive discovery finished")
		} else {
			klog.V(3).Infof("Enable dynamic drive change management using '--dynamic-drive-discovery' flag")
			klog.V(3).Infof("This flag will be made default in the next major release version")
		}

		nodeSrv, err = node.NewNodeServer(ctx, identity, nodeID, rack, zone, region, dynamicDriveDiscovery, reflinkSupport, loopbackOnly, metricsPort)
		if err != nil {
			return err
		}
		klog.V(3).Infof("node server started")

		// Check if the volume objects are migrated and CRDs versions are in-sync
		volume.SyncVolumes(ctx, nodeID)
		klog.V(3).Infof("Volumes sync completed")
	}

	var ctrlServer csi.ControllerServer
	if controller {
		ctrlServer, err = ctrl.NewControllerServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		klog.V(3).Infof("controller manager started")
	}

	return grpc.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv)
}
