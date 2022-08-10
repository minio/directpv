// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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
	"fmt"
	"net"
	"net/http"

	"k8s.io/klog/v2"
)

const (
	readinessPath = "/ready"
	readinessPort = "30443"
)

func serveReadinessEndpoint(ctx context.Context) error {
	// define http server and server handler
	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc(readinessPath, readinessHandler)
	server.Handler = mux

	lc := net.ListenConfig{}
	listener, lErr := lc.Listen(ctx, "tcp", fmt.Sprintf(":%v", readinessPort))
	if lErr != nil {
		return lErr
	}

	go func() {
		klog.V(3).Infof("Starting to serve readiness endpoint in port: %s", readinessPort)
		if err := server.Serve(listener); err != nil {
			klog.Errorf("Failed to serve readiness endpoint: %v", err)
		}
	}()

	return nil
}

// readinessHandler - Checks if the process is up. Always returns success.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("readiness request: %v", r)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}
