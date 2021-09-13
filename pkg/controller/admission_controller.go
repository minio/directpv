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

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"k8s.io/klog/v2"
)

const (
	port              = "30443"
	certPath          = "/etc/certs/cert.pem"
	keyPath           = "/etc/certs/key.pem"
	driveHandlerPath  = "/validatedrive"
	volumeHandlerPath = "/validatevolume"
)

func serveAdmissionController(ctx context.Context) {
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		klog.Errorf("Filed to load key pair: %v", err)
	}

	// Create a secure http server
	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certs},
			InsecureSkipVerify: true,
		},
	}

	// define http server and server handler
	vh := ValidationHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc(driveHandlerPath, vh.validateDrive)
	mux.HandleFunc(volumeHandlerPath, vh.validateVolume)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", port))
	if lErr != nil {
		panic(lErr)
	}

	klog.V(2).Infof("Starting admission webhook server in port: %s", port)
	if err := server.ServeTLS(listener, "", ""); err != nil {
		klog.Errorf("Failed to listen and serve admission webhook server: %v", err)
		panic(err)
	}
}
