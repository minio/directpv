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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	ctrl "github.com/minio/direct-csi/pkg/controller"
	"github.com/minio/direct-csi/pkg/converter"
	id "github.com/minio/direct-csi/pkg/identity"
	"github.com/minio/direct-csi/pkg/node"
	"github.com/minio/direct-csi/pkg/node/discovery"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils/grpc"
	"github.com/minio/direct-csi/pkg/volume"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

const (
	conversionCAFile = "/etc/conversion/CAs/ca.pem"
)

var (
	errInvalidConversionHealthzURL = errors.New("The `--conversion-webhook-healthz-url` flag is unset/empty")
)

func waitForConversionWebhook() error {
	if conversionHealthzURL == "" {
		return errInvalidConversionHealthzURL
	}

	caCert, err := ioutil.ReadFile(conversionCAFile)
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

func mkdir(dir string) (err error) {
	if err = os.Mkdir(sys.DirectCSIDevRoot, os.ModePerm); err == nil {
		return nil
	}

	if !errors.Is(err, os.ErrExist) {
		klog.Errorf("Unable to create directory %v; %v", dir, err)
		return err
	}

	checkFile := path.Join(sys.DirectCSIDevRoot, ".writecheck")
	if err = os.WriteFile(checkFile, []byte{}, os.ModePerm); err == nil {
		err = os.Remove(checkFile)
	}
	if err != nil {
		klog.Errorf("Unable to do write check on %v; %v", dir, err)
	}

	return err
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
		if err := mkdir(sys.DirectCSIDevRoot); err != nil {
			return err
		}
		if err := mkdir(sys.MountRoot); err != nil {
			return err
		}
		discovery, err := discovery.NewDiscovery(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		if err := discovery.Init(ctx, loopBackOnly); err != nil {
			return fmt.Errorf("Error while initializing drive discovery: %v", err)
		}
		klog.V(3).Infof("Drive discovery finished")

		// Check if the volume objects are migrated and CRDs versions are in-sync
		volume.SyncVolumes(ctx, nodeID)
		klog.V(3).Infof("Volumes sync completed")

		nodeSrv, err = node.NewNodeServer(ctx, identity, nodeID, rack, zone, region, enableDynamicDiscovery)
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

	return grpc.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv)
}
