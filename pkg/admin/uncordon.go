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

// UncordonArgs represents the args for uncordoning a drive
type UncordonArgs = CordonArgs

// UncordonResult represents the uncordoned drive
type UncordonResult = CordonResult

// Uncordon makes the drive schedulable again
func (client *Client) Uncordon(ctx context.Context, args UncordonArgs, log LogFunc) (results []UncordonResult, err error) {
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

		if !result.Drive.IsUnschedulable() {
			continue
		}

		result.Drive.Schedulable()
		if !args.DryRun {
			_, err = client.Drive().Update(ctx, &result.Drive, metav1.UpdateOptions{})
		}
		if err != nil {
			err = fmt.Errorf("unable to uncordon drive %v; %w", result.Drive.GetDriveID(), err)
			return
		}

		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "drive uncordoned",
				Values:           map[string]any{"node": result.Drive.GetNodeID(), "driveName": result.Drive.GetDriveName()},
				FormattedMessage: fmt.Sprintf("Drive %v/%v uncordoned\n", result.Drive.GetNodeID(), result.Drive.GetDriveName()),
			},
		)

		results = append(results, UncordonResult{
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
