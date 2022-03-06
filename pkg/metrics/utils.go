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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/fs/xfs"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	tenantLabel = "direct.csi.min.io/tenant"
)

type xfsVolumeStats struct {
	AvailableBytes uint64
	TotalBytes     uint64
	UsedBytes      uint64
}

type xfsVolumeStatsGetter func(context.Context, *directcsi.DirectCSIVolume) (xfsVolumeStats, error)

func (c *metricsCollector) getxfsVolumeStats(ctx context.Context, vol *directcsi.DirectCSIVolume) (xfsVolumeStats, error) {
	directCSIClient := client.GetDirectCSIClient()
	drive, err := directCSIClient.DirectCSIDrives().Get(ctx, vol.Status.Drive, metav1.GetOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if err != nil {
		return xfsVolumeStats{}, err
	}
	device, err := sys.GetDeviceName(drive.Status.MajorNumber, drive.Status.MinorNumber)
	if err != nil {
		return xfsVolumeStats{}, err
	}
	quota, err := xfs.GetQuota(ctx, "/dev/"+device, vol.Name)
	if err != nil {
		return xfsVolumeStats{}, err
	}
	return xfsVolumeStats{
		AvailableBytes: uint64(vol.Status.TotalCapacity) - quota.CurrentSpace,
		TotalBytes:     uint64(vol.Status.TotalCapacity),
		UsedBytes:      quota.CurrentSpace,
	}, nil
}

func publishVolumeStats(ctx context.Context, vol *directcsi.DirectCSIVolume, ch chan<- prometheus.Metric, xfsStatsFn xfsVolumeStatsGetter) {
	volStats, err := xfsStatsFn(ctx, vol)
	if err != nil {
		klog.V(3).Infof("Error while getting xfs volume stats: %v", err)
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
			"Total number of bytes used by the volume",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(volStats.UsedBytes), string(tenantName), vol.Name, vol.Status.NodeName,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName("directcsi", "stats", "bytes_total"),
			"Total number of bytes allocated to the volume",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(volStats.TotalBytes), string(tenantName), vol.Name, vol.Status.NodeName,
	)
}
