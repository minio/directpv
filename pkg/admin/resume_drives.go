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

// ResumeDriveArgs represents the args to be passed for resuming the drive
type ResumeDriveArgs = SuspendDriveArgs

// ResumeDriveResult represents the resumed drive
type ResumeDriveResult = SuspendDriveResult

// ResumeDrives will resume the suspended drives
func (client *Client) ResumeDrives(ctx context.Context, args ResumeDriveArgs, log LogFunc) (results []ResumeDriveResult, err error) {
	if log == nil {
		log = nullLogger
	}

	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(directpvtypes.ToLabelValues(args.Nodes)).
		DriveNameSelector(directpvtypes.ToLabelValues(args.Drives)).
		DriveIDSelector(args.DriveIDSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
		}
		processed = true
		if !result.Drive.IsSuspended() {
			// only suspended drives can be resumed.
			continue
		}
		driveClient := client.Drive()
		updateFunc := func() error {
			drive, err := driveClient.Get(ctx, result.Drive.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			drive.Resume()
			if !args.DryRun {
				if _, err := driveClient.Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			err = fmt.Errorf("unable to resume drive %v; %w", result.Drive.GetDriveID(), err)
			return
		}

		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "drive resumed",
				Values:           map[string]any{"node": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
				FormattedMessage: fmt.Sprintf("Drive %v/%v resumed\n", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			},
		)

		results = append(results, ResumeDriveResult{
			NodeID:    result.Drive.GetNodeID(),
			DriveName: result.Drive.GetDriveName(),
			DriveID:   result.Drive.GetDriveID(),
			Volumes:   result.Drive.GetVolumes(),
		})
	}
	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	return
}
