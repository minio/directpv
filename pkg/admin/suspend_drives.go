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
	"errors"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// ErrNoMatchingResourcesFound denotes that no matching resources are found for processing
var ErrNoMatchingResourcesFound = errors.New("no matching resources found")

// SuspendDriveArgs denotes the args for suspending the drive
type SuspendDriveArgs struct {
	Nodes            []string
	Drives           []string
	DriveIDSelectors []directpvtypes.DriveID
	DryRun           bool
}

// SuspendDriveResult represents the suspended drive
type SuspendDriveResult struct {
	NodeID    directpvtypes.NodeID
	DriveName directpvtypes.DriveName
	DriveID   directpvtypes.DriveID
	Volumes   []string
}

// SuspendDrives suspends the drive
func (client *Client) SuspendDrives(ctx context.Context, args SuspendDriveArgs, log LogFunc) (results []SuspendDriveResult, err error) {
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

		if result.Drive.IsSuspended() {
			continue
		}

		driveClient := client.Drive()
		updateFunc := func() error {
			drive, err := driveClient.Get(ctx, result.Drive.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			drive.Suspend()
			if !args.DryRun {
				if _, err := driveClient.Update(ctx, drive, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}
		if err = retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			err = fmt.Errorf("unable to suspend drive %v; %w", result.Drive.GetDriveID(), err)
			return
		}

		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "drive suspended",
				Values:           map[string]any{"node": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
				FormattedMessage: fmt.Sprintf("Drive %v/%v suspended\n", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			},
		)

		results = append(results, SuspendDriveResult{
			NodeID:    result.Drive.GetNodeID(),
			DriveName: result.Drive.GetDriveName(),
			DriveID:   result.Drive.GetDriveID(),
			Volumes:   result.Drive.GetVolumes(),
		})
	}

	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}

	return results, nil
}
