// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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

package volume

import (
	"context"
	"time"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// RunHealthMonitor periodically checks for volume health and updates the condition if the volume is in error state.
func RunHealthMonitor(ctx context.Context, nodeID directpvtypes.NodeID, interval time.Duration) error {
	healthCheckTicker := time.NewTicker(interval)
	defer healthCheckTicker.Stop()
	for {
		select {
		case <-healthCheckTicker.C:
			if err := checkVolumesHealth(ctx, nodeID, getMountpointsByVolumeName); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func checkVolumesHealth(ctx context.Context, nodeID directpvtypes.NodeID, getVolumeMounts func(string) utils.StringSet) error {
	volumes, err := NewLister().NodeSelector([]directpvtypes.LabelValue{directpvtypes.ToLabelValue(string(nodeID))}).Get(ctx)
	if err != nil {
		return err
	}
	for _, volume := range volumes {
		if !volume.IsStaged() && !volume.IsPublished() {
			continue
		}
		checkVolumeHealth(ctx, volume.Name, getVolumeMounts)
	}
	return nil
}

func checkVolumeHealth(ctx context.Context, volumeName string, getVolumeMounts func(string) utils.StringSet) {
	volume, err := client.VolumeClient().Get(
		ctx, volumeName, metav1.GetOptions{TypeMeta: types.NewVolumeTypeMeta()},
	)
	if err != nil {
		klog.V(5).ErrorS(err, "unable to get the volume", "volume", volumeName)
		return
	}
	if err = checkVolumeMounts(ctx, volume, getVolumeMounts); err != nil {
		klog.V(5).ErrorS(err, "unable to check the volume mounts", "volume", volumeName)
		return
	}
	return
}

func checkVolumeMounts(ctx context.Context, volume *types.Volume, getVolumeMounts func(string) utils.StringSet) (err error) {
	var message string
	mountExists := true
	reason := string(directpvtypes.VolumeConditionReasonNoError)
	mountPoints := getVolumeMounts(volume.Name)
	if volume.IsPublished() && !mountPoints.Exist(volume.Status.TargetPath) {
		mountExists = false
		message = string(directpvtypes.VolumeConditionMessageTargetPathNotMounted)
	}
	if volume.IsStaged() && !mountPoints.Exist(volume.Status.StagingTargetPath) {
		mountExists = false
		message = string(directpvtypes.VolumeConditionMessageStagingPathNotMounted)
	}
	if !mountExists {
		reason = string(directpvtypes.VolumeConditionReasonNotMounted)
	}
	if updatedConditions, updated := k8s.UpdateCondition(
		volume.Status.Conditions,
		string(directpvtypes.VolumeConditionTypeError),
		k8s.BoolToConditionStatus(!mountExists),
		reason,
		message,
	); updated {
		volume.Status.Conditions = updatedConditions
		_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
	}

	return
}

func getMountpointsByVolumeName(volumeName string) utils.StringSet {
	_, _, _, rootMountMap, err := sys.GetMounts(false)
	if err != nil {
		klog.V(5).ErrorS(err, "unable to get mountpoints by volume name", "volume name", volumeName)
		return nil
	}
	return rootMountMap["/"+volumeName]
}
