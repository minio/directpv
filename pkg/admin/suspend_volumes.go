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

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// SuspendVolumeArgs denotes the args for suspending the volume
type SuspendVolumeArgs struct {
	Nodes         []string
	Drives        []string
	PodNames      []string
	PodNamespaces []string
	VolumeNames   []string
	DryRun        bool
	Quiet         bool
}

// SuspendVolumes suspends the volume
func SuspendVolumes(ctx context.Context, args SuspendVolumeArgs) error {
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
			return result.Err
		}
		processed = true
		if result.Volume.IsSuspended() {
			continue
		}
		volumeClient := client.VolumeClient()
		updateFunc := func() error {
			volume, err := volumeClient.Get(ctx, result.Volume.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			volume.Suspend()
			if !args.DryRun {
				if _, err := volumeClient.Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return fmt.Errorf("unable to suspend volume %v; %v", result.Volume.Name, err)
		}
		if !args.Quiet {
			fmt.Printf("Volume %v/%v suspended\n", result.Volume.GetNodeID(), result.Volume.Name)
		}
	}
	if !processed {
		return ErrNoMatchingResourcesFound
	}
	return nil
}
