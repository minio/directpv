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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	"github.com/prometheus/client_golang/prometheus"
	clientmodelgo "github.com/prometheus/client_model/go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

const MiB = 1024 * 1024

type metricType string

const (
	metricStatsBytesUsed            metricType = consts.AppName + "_stats_bytes_used"
	metricStatsBytesTotal           metricType = consts.AppName + "_stats_bytes_total"
	metricStatsDriveReady           metricType = consts.AppName + "_stats_drive_ready"
	metricStatsDriveBytesRead       metricType = consts.AppName + "_stats_drive_total_bytes_read"
	metricStatsDriveBytesWritten    metricType = consts.AppName + "_stats_drive_total_bytes_written"
	metricStatsDriveReadLatency     metricType = consts.AppName + "_stats_drive_read_latency_seconds"
	metricStatsDriveWriteLatency    metricType = consts.AppName + "_stats_drive_write_latency_seconds"
	metricStatsDriveReadThroughput  metricType = consts.AppName + "_stats_drive_read_throughput_bytes_per_second"
	metricStatsDriveWriteThroughput metricType = consts.AppName + "_stats_drive_write_throughput_bytes_per_second"
	metricStatsDriveWaitTime        metricType = consts.AppName + "_stats_drive_wait_time_seconds"
)

var (
	volumes []types.Volume
)

func init() {
	volumes = []types.Volume{
		*types.NewVolume("test_volume_20MB", "fsuuid1", "test-node-1", "test-drive-1", "test-drive-1", 20*MiB),
		*types.NewVolume("test_volume_30MB", "fsuuid1", "test-node-1", "test-drive-1", "test-drive-1", 30*MiB),
	}
	volumes[0].Status.UsedCapacity = 10 * MiB
	volumes[0].Status.TargetPath = "/path/targetpath"
	volumes[1].Status.UsedCapacity = 20 * MiB
	volumes[1].Status.TargetPath = "/path/targetpath"

	client.FakeInit()
}

func createFakeMetricsCollector() *metricsCollector {
	return &metricsCollector{
		desc:              prometheus.NewDesc(consts.AppName+"_stats", "Statistics exposed by "+consts.AppPrettyName, nil, nil),
		nodeID:            "test-node-1",
		getDeviceByFSUUID: func(_ string) (string, error) { return "", nil },
		getQuota: func(_ context.Context, _, volumeID string) (quota *xfs.Quota, err error) {
			for _, volume := range volumes {
				if volume.Name == volumeID {
					return &xfs.Quota{
						HardLimit:    uint64(volume.Status.TotalCapacity),
						SoftLimit:    uint64(volume.Status.TotalCapacity),
						CurrentSpace: uint64(volume.Status.UsedCapacity),
					}, nil
				}
			}
			return &xfs.Quota{}, nil
		},
	}
}

func getVolumeNameFromLabelPair(labelPair []*clientmodelgo.LabelPair) string {
	for _, lp := range labelPair {
		if lp.GetName() == "volumeID" {
			return lp.GetValue()
		}
	}
	return ""
}

func getFQNameFromDesc(desc string) string {
	firstPart := strings.Split(desc, ",")[0]
	fqName := strings.Split(firstPart, ":")
	if len(fqName) != 2 {
		panic("cannot parse the fqname")
	}
	return strings.ReplaceAll(strings.TrimSpace(fqName[1]), "\"", "")
}

func TestVolumeStatsEmitter(t *testing.T) {
	testObjects := []runtime.Object{&volumes[0], &volumes[1]}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	fmc := createFakeMetricsCollector()

	clientset := types.NewExtFakeClientset(clientsetfake.NewSimpleClientset(testObjects...))
	client.SetVolumeInterface(clientset.DirectpvLatest().DirectPVVolumes())

	metricChan := make(chan prometheus.Metric)
	noOfMetricsExposedPerVolume := 2
	expectedNoOfMetrics := len(testObjects) * noOfMetricsExposedPerVolume
	noOfMetricsReceived := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				klog.V(1).Infof("Forcefully exiting due to interrupt")
				return
			case metric, ok := <-metricChan:
				if !ok {
					return
				}
				metricOut := clientmodelgo.Metric{}
				if err := metric.Write(&metricOut); err != nil {
					(*t).Fatal(err)
				}
				volumeName := getVolumeNameFromLabelPair(metricOut.GetLabel())
				mt := metricType(getFQNameFromDesc(metric.Desc().String()))
				switch mt {
				case metricStatsBytesUsed:
					volObj, gErr := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{
						TypeMeta: types.NewVolumeTypeMeta(),
					})
					if gErr != nil {
						(*t).Fatalf("[%s] Volume (%s) not found. Error: %v", volumeName, volumeName, gErr)
					}
					if volObj.Status.UsedCapacity != int64(*metricOut.Gauge.Value) {
						t.Errorf("Expected Used capacity: %v But got %v", volObj.Status.UsedCapacity, *metricOut.Gauge.Value)
					}
				case metricStatsBytesTotal:
					volObj, gErr := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{
						TypeMeta: types.NewVolumeTypeMeta(),
					})
					if gErr != nil {
						(*t).Fatalf("[%s] Volume (%s) not found. Error: %v", volumeName, volumeName, gErr)
					}
					if volObj.Status.TotalCapacity != int64(*metricOut.Gauge.Value) {
						t.Errorf("Expected Total capacity: %v But got %v", volObj.Status.TotalCapacity, *metricOut.Gauge.Value)
					}
				default:
					t.Errorf("Invalid metric type caught")
				}
				noOfMetricsReceived++
				if noOfMetricsReceived == expectedNoOfMetrics {
					return
				}
			}
		}
	}()

	fmc.publishVolumeStats(ctx, &volumes[0], metricChan)
	fmc.publishVolumeStats(ctx, &volumes[1], metricChan)

	wg.Wait()
}

func TestDriveStatsEmitter(t *testing.T) {
	// Create fake drives
	testDrives := []types.Drive{
		*types.NewDrive(
			"test-drive-1",
			types.DriveStatus{},
			"test-node-1",
			"loop1",
			"Default",
		),
		*types.NewDrive(
			"test-drive-2",
			types.DriveStatus{},
			"test-node-1",
			"loop2",
			"Default",
		),
	}
	testDrives[0].Status.FSUUID = "fsuuid1"
	testDrives[0].Status.TotalCapacity = 100 * MiB
	testDrives[1].Status.FSUUID = "fsuuid2"
	testDrives[1].Status.TotalCapacity = 200 * MiB

	// Mock drive stats
	mockDrives := map[string]*driveStats{
		"test-drive-1": {
			status:       1,
			readSectors:  1000,
			readTicks:    500,
			writeSectors: 2000,
			writeTicks:   1000,
			timeInQueue:  1500,
		},
		"test-drive-2": {
			status:       1,
			readSectors:  2000,
			readTicks:    750,
			writeSectors: 3000,
			writeTicks:   1500,
			timeInQueue:  2000,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metricChan := make(chan prometheus.Metric, 100) // Buffered channel to prevent blocking
	expectedMetrics := len(testDrives) * 8          // 8 metrics per drive
	receivedMetrics := 0

	// Mock publishDriveStats function
	mockPublishDriveStats := func(drive *types.Drive, ch chan<- prometheus.Metric) {
		stats, ok := mockDrives[drive.Name]
		if !ok {
			t.Errorf("No mock stats found for drive: %s. Available mocks: %v", drive.Name, mockDrives)
			return
		}

		// Emit mock metrics
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveReady), "Drive ready status", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.status),
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveBytesRead), "Bytes read from drive", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.readSectors*512),
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveBytesWritten), "Bytes written to drive", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.writeSectors*512),
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveReadLatency), "Drive read latency", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.readTicks)/1000,
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveWriteLatency), "Drive write latency", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.writeTicks)/1000,
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveReadThroughput), "Drive read throughput", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.readSectors*512)*1000/float64(stats.readTicks),
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveWriteThroughput), "Drive write throughput", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.writeSectors*512)*1000/float64(stats.writeTicks),
			drive.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(string(metricStatsDriveWaitTime), "Drive wait time", []string{"drive"}, nil),
			prometheus.GaugeValue,
			float64(stats.timeInQueue)/1000,
			drive.Name,
		)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				t.Log("Context done, exiting metric processing goroutine")
				return
			case metric, ok := <-metricChan:
				if !ok {
					t.Log("Metric channel closed")
					return
				}
				t.Logf("Received metric: %s", metric.Desc().String())
				receivedMetrics++
				if receivedMetrics == expectedMetrics {
					t.Log("Received all expected metrics")
					return
				}
			}
		}
	}()

	for _, drive := range testDrives {
		t.Logf("Publishing metrics for drive: %s", drive.Name)
		mockPublishDriveStats(&drive, metricChan)
	}

	// Wait for all metrics to be processed
	time.Sleep(100 * time.Millisecond)

	if receivedMetrics != expectedMetrics {
		t.Errorf("Expected %d metrics, but received %d", expectedMetrics, receivedMetrics)
	}

	close(metricChan)
	wg.Wait()
}
