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

package metrics

import (
	"context"
	"fmt"
	"net"
	"net/http"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

func metricsHandler(nodeID directpvtypes.NodeID) http.Handler {
	mc := newMetricsCollector(nodeID)
	prometheus.MustRegister(mc)

	registry := prometheus.NewRegistry()
	if err := registry.Register(mc); err != nil {
		panic(err)
	}

	gatherers := prometheus.Gatherers{
		registry,
	}

	return promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
}

// ServeMetrics starts metrics service.
func ServeMetrics(ctx context.Context, nodeID directpvtypes.NodeID, port int) {
	config := net.ListenConfig{}
	listener, err := config.Listen(ctx, "tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		panic(err)
	}

	server := &http.Server{Handler: metricsHandler(nodeID)}

	klog.V(2).Infof("Starting metrics exporter at port %v", port)
	if err := server.Serve(listener); err != nil {
		klog.ErrorS(err, "unable to start metrics server")
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}
