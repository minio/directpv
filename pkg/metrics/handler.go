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
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newMetricsCollector(nodeID string) *metricsCollector {
	mc := &metricsCollector{
		desc:   prometheus.NewDesc("directcsi_stats", "Statistics exposed by DirectCSI", nil, nil),
		nodeID: nodeID,
	}
	prometheus.MustRegister(mc)
	return mc
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
	volumeStatsEmitter(c.nodeID, ch)
}

func volumeStatsEmitter(nodeID string, ch chan<- prometheus.Metric) {
	volumeClient := utils.GetDirectCSIClient().DirectCSIVolumes()
	volumeList, err := volumeClient.List(context.Background(), metav1.ListOptions{
		TypeMeta: utils.DirectCSIVolumeTypeMeta(strings.Join([]string{directcsi.Group, directcsi.Version}, "/")),
	})
	if err != nil {
		glog.V(3).Infof("Error while listing DirectCSI Volumes: %v", err)
		return
	}
	volumes := volumeList.Items
	for _, volume := range volumes {
		isVolumePublished := func() bool {
			if volume.Status.ContainerPath != "" {
				return true
			}
			return false
		}
		// Skip volumes from other nodes
		if volume.Status.NodeName != nodeID || !isVolumePublished() {
			continue
		}
		publishVolumeStats(&volume, ch)
	}
}

func metricsHandler(nodeID string) http.Handler {

	registry := prometheus.NewRegistry()

	if err := registry.Register(newMetricsCollector(nodeID)); err != nil {
		panic(err)
	}

	gatherers := prometheus.Gatherers{
		// prometheus.DefaultGatherer,
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
