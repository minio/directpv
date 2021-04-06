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
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	ctrl "github.com/minio/direct-csi/pkg/controller"
	"github.com/minio/direct-csi/pkg/converter"
	id "github.com/minio/direct-csi/pkg/identity"
	"github.com/minio/direct-csi/pkg/node"
	"github.com/minio/direct-csi/pkg/utils"
	"github.com/minio/direct-csi/pkg/utils/grpc"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"github.com/minio/minio/pkg/ellipses"
)

const (
	caFile = "/etc/CAs/ca.pem"
)

var (
	conversionHookURLPollInterval  = 3 * time.Second
	errInvalidConversionWebhookURL = errors.New("The `--conversion-webhook-url` flag is unset/empty")
)

func waitForConversionWebhook() error {
	if conversionWebhookURL == "" {
		return errInvalidConversionWebhookURL
	}

	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		glog.V(2).Infof("Error while reading cacert %v", err)
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
	}
	defer client.CloseIdleConnections()

	for {
		_, err := client.Get(conversionWebhookURL)
		if err == nil {
			glog.V(2).Info("The conversion webhook is live!")
			// The conversion webhook is live
			break
		}
		glog.V(2).Infof("Waiting for conversion webhook: %v", err)
		time.Sleep(conversionHookURLPollInterval)
	}

	return nil
}

func run(ctx context.Context, args []string) error {

	if conversionWebhook {
		// Start conversion webserver
		if err := converter.ServeConversionWebhook(ctx); err != nil {
			return err
		}
		// Do not start node server and central controller in conversion mode
		return nil
	}

	if err := waitForConversionWebhook(); err != nil {
		return err
	}

	utils.Init()
	idServer, err := id.NewIdentityServer(identity, Version, map[string]string{})
	if err != nil {
		return err
	}
	glog.V(5).Infof("identity server started")

	basePaths := []string{}
	for _, a := range args {
		if ellipses.HasEllipses(a) {
			p, err := ellipses.FindEllipsesPatterns(a)
			if err != nil {
				return err
			}
			patterns := p.Expand()
			for _, outer := range patterns {
				basePaths = append(basePaths, strings.Join(outer, ""))
			}
		} else {
			basePaths = append(basePaths, a)
		}
	}

	var nodeSrv csi.NodeServer

	if driver {
		nodeSrv, err = node.NewNodeServer(ctx, identity, nodeID, rack, zone, region, basePaths, procfs, loopBackOnly)
		if err != nil {
			return err
		}
		glog.V(5).Infof("node server started")
	}

	var ctrlServer csi.ControllerServer
	if controller {
		ctrlServer, err = ctrl.NewControllerServer(ctx, identity, nodeID, rack, zone, region)
		if err != nil {
			return err
		}
		glog.V(5).Infof("controller manager started")
	}

	return grpc.Run(ctx, endpoint, idServer, ctrlServer, nodeSrv)
}
