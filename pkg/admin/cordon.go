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
)

// CordonResult represents the
type CordonResult struct {
	NodeID    directpvtypes.NodeID
	DriveName directpvtypes.DriveName
	DriveID   directpvtypes.DriveID
}

// CordonArgs represents the args to Cordon the drive
type CordonArgs struct {
	Nodes    []string
	Drives   []string
	Status   []directpvtypes.DriveStatus
	DriveIDs []directpvtypes.DriveID
	DryRun   bool
}

// Cordon makes a drive unschedulable
func (client *Client) Cordon(ctx context.Context, args CordonArgs, log LogFunc) (results []CordonResult, err error) {
	if log == nil {
		log = nullLogger
	}

	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(directpvtypes.ToLabelValues(args.Nodes)).
		DriveNameSelector(directpvtypes.ToLabelValues(args.Drives)).
		StatusSelector(args.Status).
		DriveIDSelector(args.DriveIDs).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
		}

		processed = true

		if result.Drive.IsUnschedulable() {
			continue
		}

		volumes := result.Drive.GetVolumes()
		if len(volumes) != 0 {
			for vresult := range client.NewVolumeLister().VolumeNameSelector(volumes).IgnoreNotFound(true).List(ctx) {
				if vresult.Err != nil {
					err = vresult.Err
					return
				}

				if vresult.Volume.Status.Status == directpvtypes.VolumeStatusPending {
					err = fmt.Errorf("unable to cordon drive %v; pending volumes found", result.Drive.GetDriveID())
					return
				}
			}
		}

		result.Drive.Unschedulable()
		if !args.DryRun {
			if _, err = client.Drive().Update(ctx, &result.Drive, metav1.UpdateOptions{}); err != nil {
				err = fmt.Errorf("unable to cordon drive %v; %w", result.Drive.GetDriveID(), err)
				return
			}
		}

		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "drive cordoned",
				Values:           map[string]any{"nodeId": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
				FormattedMessage: fmt.Sprintf("Drive %v/%v cordoned\n", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			},
		)

		results = append(results, CordonResult{
			NodeID:    result.Drive.GetNodeID(),
			DriveName: result.Drive.GetDriveName(),
			DriveID:   result.Drive.GetDriveID(),
		})
	}

	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	return
}
