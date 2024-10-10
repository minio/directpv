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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/xfs"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

const defaultSectorSize = 512

type driveStats struct {
	readBytes   float64
	readTicks   float64
	writeBytes  float64
	writeTicks  float64
	timeInQueue float64
}

func getDriveStats(driveName string) (*driveStats, error) {
	stat, err := device.GetStat(driveName)
	switch {
	case err != nil:
		return nil, err
	case len(stat) == 0:
		return nil, fmt.Errorf("unable to read stat from drive %v", driveName)
	case len(stat) < 10:
		return nil, fmt.Errorf("invalid stat format from drive %v", driveName)
	}

	// Refer https://www.kernel.org/doc/Documentation/block/stat.txt for meaning of each field.
	return &driveStats{
		readBytes:   float64(stat[2] * defaultSectorSize),
		readTicks:   float64(stat[3]),
		writeBytes:  float64(stat[6] * defaultSectorSize),
		writeTicks:  float64(stat[7]),
		timeInQueue: float64(stat[10]),
	}, nil
}

type metricsCollector struct {
	nodeID            directpvtypes.NodeID
	desc              *prometheus.Desc
	getDeviceByFSUUID func(fsuuid string) (string, error)
	getQuota          func(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error)
}

func newMetricsCollector(nodeID directpvtypes.NodeID) *metricsCollector {
	return &metricsCollector{
		nodeID:            nodeID,
		desc:              prometheus.NewDesc(consts.AppName+"_stats", "Statistics exposed by "+consts.AppPrettyName, nil, nil),
		getDeviceByFSUUID: sys.GetDeviceByFSUUID,
		getQuota:          xfs.GetQuota,
	}
}

// Describe sends the super set of all possible descriptors of metrics
func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *metricsCollector) publishVolumeStats(ctx context.Context, volume *types.Volume, ch chan<- prometheus.Metric) {
	device, err := c.getDeviceByFSUUID(volume.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				"on the host to reload",
			"FSUUID", volume.Status.FSUUID)
		client.Eventf(
			volume, client.EventTypeWarning, client.EventReasonMetrics,
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", volume.Status.FSUUID)
		return
	}
	quota, err := c.getQuota(ctx, device, volume.Name)
	if err != nil {
		klog.ErrorS(err, "unable to get quota information", "volume", volume.Name)
		return
	}

	tenantName := volume.GetTenantName()

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "bytes_used"),
			"Total number of bytes used by the volume",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(quota.CurrentSpace), tenantName, volume.Name, string(volume.GetNodeID()),
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "bytes_total"),
			"Total number of bytes allocated to the volume",
			[]string{"tenant", "volumeID", "node"}, nil),
		prometheus.GaugeValue,
		float64(volume.Status.TotalCapacity), tenantName, volume.Name, string(volume.GetNodeID()),
	)
}

func (c *metricsCollector) publishDriveStats(drive *types.Drive, ch chan<- prometheus.Metric) {
	deviceID, err := c.getDeviceByFSUUID(drive.Status.FSUUID)
	if err != nil {
		klog.ErrorS(
			err,
			"unable to find device by FSUUID; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload",
			"FSUUID", drive.Status.FSUUID)
		client.Eventf(
			drive, client.EventTypeWarning, client.EventReasonMetrics,
			"unable to find device by FSUUID %v; "+
				"either device is removed or run command "+
				"`sudo udevadm control --reload-rules && sudo udevadm trigger`"+
				" on the host to reload", drive.Status.FSUUID)

		return
	}
	deviceName := utils.TrimDevPrefix(deviceID)

	status := float64(1) // Online
	driveStat, err := getDriveStats(deviceName)
	if err != nil {
		klog.ErrorS(err, "unable to read drive statistics")
		status = float64(0) // Offline
	}

	// Metrics
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_ready"),
			"Drive Online/Offline Status",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		status, drive.Name,
	)

	if driveStat == nil {
		return
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_read_bytes"),
			"Total number of bytes read from the drive",
			[]string{"drive"}, nil),
		prometheus.CounterValue,
		driveStat.readBytes, drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_write_bytes"),
			"Total number of bytes written to the drive",
			[]string{"drive"}, nil),
		prometheus.CounterValue,
		driveStat.writeBytes, drive.Name,
	)

	// Drive Read/Write Latency
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_read_latency_seconds"),
			"Drive Read Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		driveStat.readTicks/1000, drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_write_latency_seconds"),
			"Drive Write Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		driveStat.writeTicks/1000, drive.Name,
	)

	// Wait Time
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_wait_time_seconds"),
			"Drive Wait Time",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		driveStat.timeInQueue/1000, drive.Name,
	)
}

// Collect is called by Prometheus registry when collecting metrics.
func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Collecting volume statistics
	volumeResultCh := client.NewVolumeLister().
		NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(c.nodeID))}).
		List(ctx)
	for result := range volumeResultCh {
		if result.Err != nil {
			break
		}

		if result.Volume.Status.TargetPath != "" {
			c.publishVolumeStats(ctx, &result.Volume, ch)
		}
	}

	// Collecting drive statistics
	driveResultCh := client.NewDriveLister().
		NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(c.nodeID))}).
		List(ctx)
	for result := range driveResultCh {
		if result.Err != nil {
			break
		}

		c.publishDriveStats(&result.Drive, ch)
	}
}
