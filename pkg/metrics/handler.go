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
	"net/http"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newMetricsCollector(nodeID string) (*metricsCollector, error) {
	mc := &metricsCollector{
		desc:   prometheus.NewDesc("directcsi_stats", "Statistics exposed by DirectCSI", nil, nil),
		nodeID: nodeID,
	}
	prometheus.MustRegister(mc)
	return mc, nil
}

type metricsCollector struct {
	desc   *prometheus.Desc
	nodeID string
}

// Describe sends the super-set of all possible descriptors of metrics
func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.volumeStatsEmitter(context.Background(), ch, c.getxfsVolumeStats)
}

func (c *metricsCollector) volumeStatsEmitter(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	volumeStatsGetter xfsVolumeStatsGetter) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh, err := client.ListVolumes(
		ctx,
		[]utils.LabelValue{utils.NewLabelValue(c.nodeID)},
		nil,
		nil,
		nil,
		client.MaxThreadCount,
	)
	if err != nil {
		klog.V(3).Infof("Error while listing DirectPV Volumes: %v", err)
		return
	}

	for result := range resultCh {
		if result.Err != nil {
			return
		}

		if result.Volume.Status.ContainerPath != "" {
			publishVolumeStats(ctx, &result.Volume, ch, volumeStatsGetter)
		}
	}
}

func metricsHandler(nodeID string) http.Handler {

	registry := prometheus.NewRegistry()

	mc, err := newMetricsCollector(nodeID)
	if err != nil {
		panic(err)
	}

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
