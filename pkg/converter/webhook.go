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

package converter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"k8s.io/klog/v2"
)

const (
	port        = "30443"
	certPath    = "/etc/conversion/certs/cert.pem"
	keyPath     = "/etc/conversion/certs/key.pem"
	healthzPath = "/healthz"

	// DriveHandlerPath denotes drive conversion URI.
	DriveHandlerPath = "/convertdrive"

	// VolumeHandlerPath denotes volume conversion URI.
	VolumeHandlerPath = "/convertvolume"
)

// ServeConversionWebhook starts conversion webhook service.
func ServeConversionWebhook(ctx context.Context) error {
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		klog.Errorf("Filed to load key pair: %v", err)
		return err
	}

	// Create a secure http server
	server := &http.Server{
		TLSConfig: &tls.Config{
			Certificates:       []tls.Certificate{certs},
			InsecureSkipVerify: true,
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc(DriveHandlerPath, serveDriveConversion)
	mux.HandleFunc(VolumeHandlerPath, serveVolumeConversion)
	mux.HandleFunc(healthzPath, LivenessCheckHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", port))
	if lErr != nil {
		return lErr
	}

	go func() {
		klog.V(3).Infof("Starting conversion webhook server in port: %s", port)
		if err := server.ServeTLS(listener, "", ""); err != nil {
			klog.Errorf("Failed to listen and serve conversion webhook server: %v", err)
		}
	}()

	return nil
}

// LivenessCheckHandler - Checks if the process is up. Always returns success.
func LivenessCheckHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("Liveness check request: %v", r)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}
