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

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	fakedirect "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"k8s.io/klog/v2"
)

const (
	testNodeName   = "test-node-1"
	testDriveName  = "test-drive-1"
	testTenantName = "tenant-1"

	KB = 1 << 10
	MB = KB << 10

	mb20 = 20 * MB
	mb30 = 30 * MB
	mb10 = 10 * MB
)

type metricType string

const (
	metricStatsBytesUsed  metricType = "directpv_stats_bytes_used"
	metricStatsBytesTotal metricType = "directpv_stats_bytes_total"
)

func createFakeMetricsCollector() *metricsCollector {
	return &metricsCollector{
		desc:   prometheus.NewDesc("directcsi_stats", "Statistics exposed by DirectCSI", nil, nil),
		nodeID: testNodeName,
	}
}

func getVolumeNameFromLabelPair(labelPair []*dto.LabelPair) string {
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
	testVolumeName20MB := "test_volume_20MB"
	testVolumeName30MB := "test_volume_30MB"

	createTestVolume := func(volName string, totalCap, usedCap int64) *directcsi.DirectCSIVolume {
		return &directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: volName,
				Labels: map[string]string{
					tenantLabel: testTenantName,
				},
			},
			Status: directcsi.DirectCSIVolumeStatus{
				NodeName:      testNodeName,
				Drive:         testDriveName,
				TotalCapacity: totalCap,
				ContainerPath: "/path/containerpath",
				UsedCapacity:  usedCap,
			},
		}
	}

	testStatsGetter := func(_ context.Context, vol *directcsi.DirectCSIVolume) (xfsVolumeStats, error) {
		return xfsVolumeStats{
			TotalBytes:     uint64(vol.Status.TotalCapacity),
			UsedBytes:      uint64(vol.Status.UsedCapacity),
			AvailableBytes: uint64(vol.Status.TotalCapacity - vol.Status.UsedCapacity),
		}, nil
	}

	testObjects := []runtime.Object{
		createTestVolume(testVolumeName20MB, mb20, mb10),
		createTestVolume(testVolumeName30MB, mb30, mb20),
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	fmc := createFakeMetricsCollector()
	client.SetLatestDirectCSIVolumeInterface(fakedirect.NewSimpleClientset(testObjects...).DirectV1beta4().DirectCSIVolumes())

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
				metricOut := dto.Metric{}
				if err := metric.Write(&metricOut); err != nil {
					(*t).Fatal(err)
				}
				volumeName := getVolumeNameFromLabelPair(metricOut.GetLabel())
				mt := metricType(getFQNameFromDesc(metric.Desc().String()))
				switch mt {
				case metricStatsBytesUsed:
					volObj, gErr := client.GetLatestDirectCSIVolumeInterface().Get(ctx, volumeName, metav1.GetOptions{
						TypeMeta: utils.DirectCSIVolumeTypeMeta(),
					})
					if gErr != nil {
						(*t).Fatalf("[%s] Volume (%s) not found. Error: %v", volumeName, volumeName, gErr)
					}
					if int64(volObj.Status.UsedCapacity) != int64(*metricOut.Gauge.Value) {
						t.Errorf("Expected Used capacity: %v But got %v", int64(volObj.Status.UsedCapacity), int64(*metricOut.Gauge.Value))
					}
				case metricStatsBytesTotal:
					volObj, gErr := client.GetLatestDirectCSIVolumeInterface().Get(ctx, volumeName, metav1.GetOptions{
						TypeMeta: utils.DirectCSIVolumeTypeMeta(),
					})
					if gErr != nil {
						(*t).Fatalf("[%s] Volume (%s) not found. Error: %v", volumeName, volumeName, gErr)
					}
					if int64(volObj.Status.TotalCapacity) != int64(*metricOut.Gauge.Value) {
						t.Errorf("Expected Total capacity: %v But got %v", int64(volObj.Status.TotalCapacity), int64(*metricOut.Gauge.Value))
					}
				default:
					t.Errorf("Invalid metric type caught")
				}
				noOfMetricsReceived = noOfMetricsReceived + 1
				if noOfMetricsReceived == expectedNoOfMetrics {
					return
				}
			}
		}
	}()

	fmc.volumeStatsEmitter(ctx, metricChan, testStatsGetter)

	wg.Wait()
	cancel()
}
