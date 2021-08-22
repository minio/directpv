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

package metrics

import (
	"context"
	"net/http"

	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/sys/fs/quota"
	"github.com/minio/direct-csi/pkg/utils"

	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newMetricsCollector(nodeID string) (*metricsCollector, error) {
	config, err := utils.GetKubeConfig()
	if err != nil {
		return &metricsCollector{}, err
	}

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return &metricsCollector{}, err
	}

	mc := &metricsCollector{
		desc:            prometheus.NewDesc("directcsi_stats", "Statistics exposed by DirectCSI", nil, nil),
		nodeID:          nodeID,
		directcsiClient: directClientset,
		quotaer:         &quota.DefaultDriveQuotaer{},
	}
	prometheus.MustRegister(mc)
	return mc, nil
}

type metricsCollector struct {
	desc            *prometheus.Desc
	nodeID          string
	directcsiClient clientset.Interface
	quotaer         quota.Quotaer
}

// Describe sends the super-set of all possible descriptors of metrics
func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.volumeStatsEmitter(context.Background(), ch, c.getXFSVolumeStats)
}

func (c *metricsCollector) volumeStatsEmitter(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	volumeStatsGetter xfsVolumeStatsGetter) {
	volumes, err := utils.GetVolumeList(context.Background(), c.directcsiClient.DirectV1beta2().DirectCSIVolumes(), nil, nil, nil, nil)
	if err != nil {
		klog.V(3).Infof("Error while listing DirectCSI Volumes: %v", err)
		return
	}
	for _, volume := range volumes {
		isVolumePublished := func() bool {
			if volume.Status.ContainerPath != "" {
				return true
			}
			return false
		}
		// Skip volumes from other nodes
		if volume.Status.NodeName != c.nodeID || !isVolumePublished() {
			continue
		}
		publishVolumeStats(ctx, &volume, ch, volumeStatsGetter)
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
