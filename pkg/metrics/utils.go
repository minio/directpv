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

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta1"
	"github.com/minio/direct-csi/pkg/sys/xfs"

	"github.com/golang/glog"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	tenantLabel = "direct.csi.min.io/tenant"
)

func publishVolumeStats(vol *directcsi.DirectCSIVolume, ch chan<- prometheus.Metric) {
	xfsQuota := &xfs.XFSQuota{
		Path:      vol.Status.StagingPath,
		ProjectID: vol.Name,
	}
	volStats, err := xfsQuota.GetVolumeStats(context.Background())
	if err != nil {
		glog.V(3).Infof("Error while getting xfs volume stats: %v", err)
		return
	}

	getTenantName := func() string {
		labels := vol.ObjectMeta.GetLabels()
		for k, v := range labels {
			if k == tenantLabel {
				return v
			}
		}
		return ""
	}
	tenantName := getTenantName()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName("directcsi", "stats", "bytes_used"),
			"Total number of bytes used",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(volStats.UsedBytes), string(tenantName), vol.Name, vol.Status.NodeName,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName("directcsi", "stats", "bytes_total"),
			"Total number of bytes allocated",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(volStats.TotalBytes), string(tenantName), vol.Name, vol.Status.NodeName,
	)

	// ch <- prometheus.MustNewConstMetric(
	// 	prometheus.NewDesc(
	// 		prometheus.BuildFQName("directcsi", "stats", "bytes_available"),
	// 		"Total number of bytes available",
	// 		[]string{"tenant", "volumeID", "node"}, nil),
	// 	prometheus.GaugeValue,
	// 	float64(volStats.AvailableBytes), string(tenantName), volume.Name, volume.Status.NodeName,
	// )
}
