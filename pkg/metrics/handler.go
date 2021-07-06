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
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func newMetricsCollector(nodeID string) (*metricsCollector, error) {
	kubeConfig := utils.GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return &metricsCollector{}, err
		}
	}

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return &metricsCollector{}, err
	}

	mc := &metricsCollector{
		desc:            prometheus.NewDesc("directcsi_stats", "Statistics exposed by DirectCSI", nil, nil),
		nodeID:          nodeID,
		directcsiClient: directClientset,
	}
	prometheus.MustRegister(mc)
	return mc, nil
}

type metricsCollector struct {
	desc            *prometheus.Desc
	nodeID          string
	directcsiClient clientset.Interface
}

// Describe sends the super-set of all possible descriptors of metrics
func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Collect is called by the Prometheus registry when collecting metrics.
func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.volumeStatsEmitter(context.Background(), ch, getXFSVolumeStats)
}

func (c *metricsCollector) volumeStatsEmitter(
	ctx context.Context,
	ch chan<- prometheus.Metric,
	volumeStatsGetter xfsVolumeStatsGetter) {
	volumeClient := c.directcsiClient.DirectV1beta2().DirectCSIVolumes()
	volumeList, err := volumeClient.List(
		context.Background(),
		metav1.ListOptions{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
		},
	)
	if err != nil {
		klog.V(3).Infof("Error while listing DirectCSI Volumes: %v", err)
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
