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
)

// UncordonArgs represents the args for uncordoning a drive
type UncordonArgs = CordonArgs

// Uncordon makes the drive schedulable again
func Uncordon(ctx context.Context, args UncordonArgs) error {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewDriveLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		StatusSelector(args.Status).
		DriveIDSelector(args.DriveIDs).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}

		processed = true

		if !result.Drive.IsUnschedulable() {
			continue
		}

		result.Drive.Schedulable()
		var err error
		if !args.DryRun {
			_, err = client.DriveClient().Update(ctx, &result.Drive, metav1.UpdateOptions{})
		}

		if err != nil {
			return fmt.Errorf("unable to uncordon drive %v; %v", result.Drive.GetDriveID(), err)
		}

		if !args.Quiet {
			fmt.Printf("Drive %v/%v uncordoned\n", result.Drive.GetNodeID(), result.Drive.GetDriveName())
		}
	}

	if !processed {
		return ErrNoMatchingResourcesFound
	}
	return nil
}
