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
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
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

type DriveStats struct {
	ReadSectors  int64
	ReadTicks    int64
	WriteSectors int64
	WriteTicks   int64
	TimeInQueue  int64
}

func (c *metricsCollector) publishDriveStats(ctx context.Context, drive *types.Drive, ch chan<- prometheus.Metric) {
	device, err := c.getDeviceByFSUUID(drive.Status.FSUUID)
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

	// Online/Offline Status
	if _, err := os.Stat("/sys/block/" + device); os.IsNotExist(err) {
		// Drive is offline
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(consts.AppName, "stats", "drive_status"),
				"Drive Online/Offline Status",
				[]string{"drive"}, nil),
			prometheus.GaugeValue,
			0, // 0 for offline
			drive.Name,
		)
	} else {
		// Drive is online
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(consts.AppName, "stats", "drive_status"),
				"Drive Online/Offline Status",
				[]string{"drive"}, nil),
			prometheus.GaugeValue,
			1, // 1 for online
			drive.Name,
		)
	}

	filePath := "/sys/block/" + device + "/stat"

	driveStat, err := readDriveStats(filePath)
	if err != nil {
		klog.ErrorS(err, "unable to read drive statistics", "FilePath", filePath)
		return
	}

	sectorSizeBytes := float64(getSectorSize(device)) // Size of a sector in bytes

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_bytes_read"),
			"Total number of bytes read from the drive",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.ReadSectors)*sectorSizeBytes, drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_total_bytes_written"),
			"Total number of bytes written to the drive",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.WriteSectors)*sectorSizeBytes, drive.Name,
	)

	// Drive Read/Write Latency
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_read_latency_seconds"),
			"Drive Read Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.ReadTicks)/1000, drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_write_latency_seconds"),
			"Drive Write Latency",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.WriteTicks)/1000, drive.Name,
	)

	// Drive Read/Write Throughput
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_read_throughput_bytes_per_second"),
			"Drive Read Throughput",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.ReadSectors)*sectorSizeBytes*1000/float64(driveStat.ReadTicks), drive.Name,
	)

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_write_throughput_bytes_per_second"),
			"Drive Write Throughput",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.WriteSectors)*sectorSizeBytes*1000/float64(driveStat.WriteTicks), drive.Name,
	)

	// Wait Time
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(consts.AppName, "stats", "drive_wait_time_seconds"),
			"Drive Wait Time",
			[]string{"drive"}, nil),
		prometheus.GaugeValue,
		float64(driveStat.TimeInQueue)/1000, drive.Name,
	)
}

func readDriveStats(filePath string) (*DriveStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening stat file: %w", err)
	}
	defer file.Close()

	return parseStats(file)
}

func parseStats(reader *os.File) (*DriveStats, error) {
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			return nil, fmt.Errorf("unexpected format in stat file")
		}

		return parseDriveStats(fields)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading stat file: %w", err)
	}

	return nil, fmt.Errorf("stat file is empty")
}

func parseDriveStats(fields []string) (*DriveStats, error) {
	var stats DriveStats
	var err error

	stats.ReadSectors, err = strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing ReadSectors: %w", err)
	}
	stats.ReadTicks, err = strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing ReadTicks: %w", err)
	}
	stats.WriteSectors, err = strconv.ParseInt(fields[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing WriteSectors: %w", err)
	}
	stats.WriteTicks, err = strconv.ParseInt(fields[7], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing WriteTicks: %w", err)
	}
	stats.TimeInQueue, err = strconv.ParseInt(fields[9], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing TimeInQueue: %w", err)
	}

	return &stats, nil
}

func getSectorSize(device string) int64 {
	// Construct the file path to the sector size file
	filePath := "/sys/block/" + device + "/queue/hw_sector_size"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		// Handle error or return a default sector size
		fmt.Println("Error opening sector size file, using default sector size: ", err)
		return 512
	}
	defer file.Close()

	// Read the sector size
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		sectorSizeStr := scanner.Text()
		sectorSize, err := strconv.ParseInt(sectorSizeStr, 10, 64)
		if err != nil {
			// Handle parsing error
			fmt.Println("Error parsing sector size, using default: ", err)
			return 512
		}
		return sectorSize
	}

	if err := scanner.Err(); err != nil {
		// Handle read error
		fmt.Println("Error reading sector size, using default: ", err)
	}

	return 512 // Default sector size
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
			continue
		}

		c.publishDriveStats(ctx, &result.Drive, ch)
	}
}
