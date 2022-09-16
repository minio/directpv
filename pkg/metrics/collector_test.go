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
	metricStatsBytesUsed  metricType = consts.AppName + "_stats_bytes_used"
	metricStatsBytesTotal metricType = consts.AppName + "_stats_bytes_total"
)

var volumes = []types.Volume{
	{
		TypeMeta: types.NewVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test_volume_20MB",
			Labels: map[string]string{tenantLabel: "tenant-1"},
		},
		Status: types.VolumeStatus{
			NodeName:      "test-node-1",
			DriveName:     "test-drive-1",
			TotalCapacity: 20 * MiB,
			ContainerPath: "/path/containerpath",
			UsedCapacity:  10 * MiB,
		},
	},
	{
		TypeMeta: types.NewVolumeTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test_volume_30MB",
			Labels: map[string]string{tenantLabel: "tenant-1"},
		},
		Status: types.VolumeStatus{
			NodeName:      "test-node-1",
			DriveName:     "test-drive-1",
			TotalCapacity: 30 * MiB,
			ContainerPath: "/path/containerpath",
			UsedCapacity:  20 * MiB,
		},
	},
}

func createFakeMetricsCollector() *metricsCollector {
	return &metricsCollector{
		desc:              prometheus.NewDesc(consts.AppName+"_stats", "Statistics exposed by "+consts.AppPrettyName, nil, nil),
		nodeID:            "test-node-1",
		getDeviceByFSUUID: func(fsuuid string) (string, error) { return "", nil },
		getQuota: func(ctx context.Context, device, volumeID string) (quota *xfs.Quota, err error) {
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
					if int64(volObj.Status.UsedCapacity) != int64(*metricOut.Gauge.Value) {
						t.Errorf("Expected Used capacity: %v But got %v", int64(volObj.Status.UsedCapacity), int64(*metricOut.Gauge.Value))
					}
				case metricStatsBytesTotal:
					volObj, gErr := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{
						TypeMeta: types.NewVolumeTypeMeta(),
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

	fmc.publishVolumeStats(ctx, &volumes[0], metricChan)
	fmc.publishVolumeStats(ctx, &volumes[1], metricChan)

	wg.Wait()
}
