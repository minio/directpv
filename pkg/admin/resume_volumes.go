// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"fmt"

	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// ResumeVolumeArgs represents the args to be passed for resuming the volume
type ResumeVolumeArgs = SuspendVolumeArgs

// ResumeVolumeResult represents the suspended volume
type ResumeVolumeResult = SuspendVolumeResult

// ResumeVolumes will resume the suspended volumes
func (client *Client) ResumeVolumes(ctx context.Context, args ResumeVolumeArgs, log logFn) (results []ResumeVolumeResult, err error) {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		PodNameSelector(utils.ToLabelValues(args.PodNames)).
		PodNSSelector(utils.ToLabelValues(args.PodNamespaces)).
		VolumeNameSelector(args.VolumeNames).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
		}
		processed = true
		if !result.Volume.IsSuspended() {
			// only suspended drives can be resumed.
			continue
		}
		volumeClient := client.Volume()
		updateFunc := func() error {
			volume, err := volumeClient.Get(ctx, result.Volume.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			volume.Resume()
			if !args.DryRun {
				if _, err := volumeClient.Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}

		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			err = fmt.Errorf("unable to resume volume %v; %v", result.Volume.Name, err)
			return
		}

		log(false, "Volume %v/%v resumed\n", result.Volume.GetNodeID(), result.Volume.Name)

		results = append(results, ResumeVolumeResult{
			NodeID:     result.Volume.GetNodeID(),
			VolumeName: result.Volume.Name,
		})
	}
	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	return
}
