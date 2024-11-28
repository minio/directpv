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
)

// RemoveArgs represents the arguments to remove a drive
type RemoveArgs struct {
	Nodes       []string
	Drives      []string
	DriveStatus []directpvtypes.DriveStatus
	DriveIDs    []directpvtypes.DriveID
	DryRun      bool
}

// RemoveResult represents the removed drive
type RemoveResult struct {
	NodeID    directpvtypes.NodeID
	DriveName directpvtypes.DriveName
}

// Remove removes the initialized drive(s)
func (client *Client) Remove(ctx context.Context, args RemoveArgs, log LogFunc) (results []RemoveResult, err error) {
	if log == nil {
		log = nullLogger
	}

	var processed bool
	var failed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(directpvtypes.ToLabelValues(args.Nodes)).
		DriveNameSelector(directpvtypes.ToLabelValues(args.Drives)).
		StatusSelector(args.DriveStatus).
		DriveIDSelector(args.DriveIDs).
		IgnoreNotFound(true).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			err = result.Err
			return
		}

		processed = true
		switch result.Drive.Status.Status {
		case directpvtypes.DriveStatusRemoved:
		default:
			volumeCount := result.Drive.GetVolumeCount()
			if volumeCount > 0 {
				failed = true
			} else {
				result.Drive.Status.Status = directpvtypes.DriveStatusRemoved
				var err error
				if !args.DryRun {
					_, err = client.Drive().Update(ctx, &result.Drive, metav1.UpdateOptions{})
				}
				if err != nil {
					failed = true
					log(
						LogMessage{
							Type:             ErrorLogType,
							Err:              err,
							Message:          "unable to remove drive",
							Values:           map[string]any{"node": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
							FormattedMessage: fmt.Sprintf("%v/%v: %v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName(), err),
						},
					)
				} else {
					log(
						LogMessage{
							Type:             InfoLogType,
							Message:          "removing drive",
							Values:           map[string]any{"node": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
							FormattedMessage: fmt.Sprintf("Removing %v/%v\n", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
						},
					)
				}
				results = append(results, RemoveResult{
					NodeID:    result.Drive.GetNodeID(),
					DriveName: result.Drive.GetDriveName(),
				})
			}
		}
	}
	if !processed {
		return nil, ErrNoMatchingResourcesFound
	}
	if failed {
		err = errors.New("unable to remove drive(s)")
		return
	}
	return
}
