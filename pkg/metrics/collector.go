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
	"os/exec"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/volume"
	"github.com/minio/directpv/pkg/xfs"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

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

type driveStats struct {
	readSectors  uint64
	readTicks    uint64
	writeSectors uint64
	writeTicks   uint64
	timeInQueue  uint64
}

func (c *metricsCollector) publishDriveStats(drive *types.Drive, ch chan<- prometheus.Metric) {
	device, err := c.getDeviceByFSUUID(drive.Status.FSUUID)

	klog.Info(device)
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(consts.AppName, "stats", "drive_ready"),
				"Drive Online/Offline Status",
				[]string{"drive"}, nil),
			prometheus.GaugeValue,
			0, // 0 for offline
			drive.Name,
		)

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

	driveStatus := 0.0
	out, err := exec.Command("lsblk", "-ndo", "NAME").Output()
	if err == nil {
		devices := strings.Split(string(out), "\n")
		for _, d := range devices {
			if d == device {
				driveStatus = 1.0
			}
		}
	}

	klog.Info(driveStatus)
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_ready"),
			"Drive Online/Offline Status",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		driveStatus, // 0 for offline, 1 for online
		drive.Name,
	)

	driveStat, err := getDriveStats(device)
	if err != nil {
		klog.ErrorS(err, "unable to read drive statistics", "device", device)
		return
	}

	sectorSizeBytes := 512

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_bytes_read"),
			"Total number of bytes read from the drive",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.readSectors)*float64(sectorSizeBytes), drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_bytes_written"),
			"Total number of bytes written to the drive",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.writeSectors)*float64(sectorSizeBytes), drive.Name,
	)

	// Drive Read/Write Latency
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_read_latency_seconds"),
			"Drive Read Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.readTicks)/1000, drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_write_latency_seconds"),
			"Drive Write Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.writeTicks)/1000, drive.Name,
	)

	// Drive Read/Write Throughput
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_read_throughput_bytes_per_second"),
			"Drive Read Throughput",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.readSectors)*float64(sectorSizeBytes)*1000/float64(driveStat.readTicks), drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_write_throughput_bytes_per_second"),
			"Drive Write Throughput",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.writeSectors)*float64(sectorSizeBytes)*1000/float64(driveStat.writeTicks), drive.Name,
	)

	// Wait Time
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_wait_time_seconds"),
			"Drive Wait Time",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.timeInQueue)/1000, drive.Name,
	)
}

func getDriveStats(driveName string) (*driveStats, error) {
	stats, err := device.GetStat(driveName)
	if err != nil {
		return nil, err
	}
	if len(stats) == 0 {
		// The drive is lost
		return nil, nil
	}

	if len(stats) < 10 {
		return nil, fmt.Errorf("unknown stats from sysfs for drive %v", driveName)
	}

	// Refer https://www.kernel.org/doc/Documentation/block/stat.txt
	// for meaning of each field.
	return &driveStats{
		readSectors:  stats[2],
		readTicks:    stats[3],
		writeSectors: stats[6],
		writeTicks:   stats[7],
		timeInQueue:  stats[9],
	}, nil
}

// Collect is called by Prometheus registry when collecting metrics.
func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	// Collecting volume statistics
	volumeResultCh := volume.NewLister().
		NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(c.nodeID))}).
		List(ctx)
	for result := range volumeResultCh {
		if result.Err != nil {
			continue
		}

		if result.Volume.Status.TargetPath != "" {
			c.publishVolumeStats(ctx, &result.Volume, ch)
		}
	}

	// Collecting drive statistics
	driveResultCh := drive.NewLister().
		NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(c.nodeID))}).
		List(ctx)
	for result := range driveResultCh {
		if result.Err != nil {
			break
		}

		c.publishDriveStats(&result.Drive, ch)
	}
}
