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

package converter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/golang/glog"
)

const (
	port             = "443"
	certPath         = "/etc/certs/cert.pem"
	keyPath          = "/etc/certs/key.pem"
	DriveHandlerPath = "/convertdrive"
	healthzPath      = "/healthz"
)

func ServeConversionWebhook(ctx context.Context) error {
	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		glog.Errorf("Filed to load key pair: %v", err)
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
	mux.HandleFunc(DriveHandlerPath, ServeConversion)
	mux.HandleFunc(healthzPath, LivenessCheckHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", port))
	if lErr != nil {
		return lErr
	}

	glog.Infof("Starting conversion webhook server in port: %s", port)
	if err := server.ServeTLS(listener, "", ""); err != nil {
		glog.Errorf("Failed to listen and serve admission webhook server: %v", err)
		return err
	}

	return nil
}

// LivenessCheckHandler - Checks if the process is up. Always returns success.
func LivenessCheckHandler(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Liveness check request: %v", r)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}
