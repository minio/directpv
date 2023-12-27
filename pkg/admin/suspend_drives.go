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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"
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
	Quiet            bool
}

// SuspendDrives suspends the drive
func SuspendDrives(ctx context.Context, args SuspendDriveArgs) error {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		DriveIDSelector(args.DriveIDSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}

		processed = true

		if result.Drive.IsSuspended() {
			continue
		}

		driveClient := client.DriveClient()
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
		if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			return fmt.Errorf("unable to suspend drive %v; %v", result.Drive.GetDriveID(), err)
		}

		if !args.Quiet {
			fmt.Printf("Drive %v/%v suspended\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
		}
	}
	if !processed {
		return ErrNoMatchingResourcesFound
	}
	return nil
}
