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

	"github.com/minio/directpv/pkg/consts"
	"k8s.io/klog/v2"
)

func serveReadinessEndpoint(ctx context.Context) error {
	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc(consts.ReadinessPath, readinessHandler)
	server.Handler = mux

	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%v", readinessPort))
	if err != nil {
		return err
	}

	for {
		klog.V(3).Infof("Serving readiness endpoint at :%v", readinessPort)
		if err = server.Serve(listener); err != nil {
			klog.ErrorS(err, "unable to serve readiness endpoint")
			return err
		}
	}
}

// readinessHandler - Checks if the process is up. Always returns success.
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("Received readiness request %v", r)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
