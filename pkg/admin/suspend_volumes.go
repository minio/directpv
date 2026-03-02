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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
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
}

// SuspendVolumeResult represents the suspended volume
type SuspendVolumeResult struct {
	NodeID       directpvtypes.NodeID
	VolumeName   string
	DriveID      directpvtypes.DriveID
	DriveName    directpvtypes.DriveName
	PodName      string
	PodNamespace string
}

// SuspendVolumes suspends the volume
func (client *Client) SuspendVolumes(ctx context.Context, args SuspendVolumeArgs, log LogFunc) (results []SuspendVolumeResult, err error) {
	if log == nil {
		log = nullLogger
	}

	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		NodeSelector(directpvtypes.ToLabelValues(args.Nodes)).
		DriveNameSelector(directpvtypes.ToLabelValues(args.Drives)).
		PodNameSelector(directpvtypes.ToLabelValues(args.PodNames)).
		PodNSSelector(directpvtypes.ToLabelValues(args.PodNamespaces)).
		VolumeNameSelector(args.VolumeNames).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
		}
		processed = true
		if result.Volume.IsSuspended() {
			continue
		}
		volumeClient := client.Volume()
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
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			err = fmt.Errorf("unable to suspend volume %v; %w", result.Volume.Name, err)
			return
		}

		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "volume suspended",
				Values:           map[string]any{"node": result.Volume.GetNodeID(), "volume": result.Volume.Name},
				FormattedMessage: fmt.Sprintf("Volume %v/%v suspended\n", result.Volume.GetNodeID(), result.Volume.Name),
			},
		)

		results = append(results, SuspendVolumeResult{
			NodeID:       result.Volume.GetNodeID(),
			VolumeName:   result.Volume.Name,
			DriveID:      result.Volume.GetDriveID(),
			DriveName:    result.Volume.GetDriveName(),
			PodName:      result.Volume.GetPodName(),
			PodNamespace: result.Volume.GetPodNS(),
		})
	}
	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	return
}
