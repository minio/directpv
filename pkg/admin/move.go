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

	"github.com/dustin/go-humanize"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MoveArgs represents the args for moving
type MoveArgs struct {
	Source      directpvtypes.DriveID
	Destination directpvtypes.DriveID
}

// Move - moves the volume references from source to destination
func (client *Client) Move(ctx context.Context, args MoveArgs, log LogFunc) error {
	if log == nil {
		log = nullLogger
	}

	if args.Source == args.Destination {
		return errors.New("source and destination drives are same")
	}

	srcDrive, err := client.Drive().Get(ctx, string(args.Source), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get source drive; %w", err)
	}

	if !srcDrive.IsUnschedulable() {
		return errors.New("source drive is not cordoned")
	}

	sourceVolumeNames := srcDrive.GetVolumes()
	if len(sourceVolumeNames) == 0 {
		return fmt.Errorf("no volumes found in source drive %v", args.Source)
	}

	var requiredCapacity int64
	var volumes []types.Volume
	for result := range client.NewVolumeLister().VolumeNameSelector(sourceVolumeNames).List(ctx) {
		if result.Err != nil {
			return result.Err
		}
		if result.Volume.IsPublished() {
			return fmt.Errorf("cannot move published volume %v", result.Volume.Name)
		}
		requiredCapacity += result.Volume.Status.TotalCapacity
		volumes = append(volumes, result.Volume)
	}

	if len(volumes) == 0 {
		return fmt.Errorf("no volumes found in source drive %v", args.Source)
	}

	destDrive, err := client.Drive().Get(ctx, string(args.Destination), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get destination drive %v; %w", args.Destination, err)
	}
	if destDrive.GetNodeID() != srcDrive.GetNodeID() {
		return fmt.Errorf("source and destination drives must be in same node; source node %v; desination node %v",
			srcDrive.GetNodeID(),
			destDrive.GetNodeID())
	}
	if !destDrive.IsUnschedulable() {
		return errors.New("destination drive is not cordoned")
	}
	if destDrive.Status.Status != directpvtypes.DriveStatusReady {
		return errors.New("destination drive is not in ready state")
	}

	if srcDrive.GetAccessTier() != destDrive.GetAccessTier() {
		return fmt.Errorf("source drive access-tier %v and destination drive access-tier %v differ",
			srcDrive.GetAccessTier(),
			destDrive.GetAccessTier())
	}

	if destDrive.Status.FreeCapacity < requiredCapacity {
		return fmt.Errorf("insufficient free capacity on destination drive; required=%v free=%v",
			humanize.Comma(requiredCapacity),
			humanize.Comma(destDrive.Status.FreeCapacity))
	}

	for _, volume := range volumes {
		if destDrive.AddVolumeFinalizer(volume.Name) {
			destDrive.Status.FreeCapacity -= volume.Status.TotalCapacity
			destDrive.Status.AllocatedCapacity += volume.Status.TotalCapacity
		}
	}
	destDrive.Status.Status = directpvtypes.DriveStatusMoving
	_, err = client.Drive().Update(
		ctx, destDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		return fmt.Errorf("unable to move volumes to destination drive; %w", err)
	}

	for _, volume := range volumes {
		log(
			LogMessage{
				Type:             InfoLogType,
				Message:          "moving volume",
				Values:           map[string]any{"volume": volume.Name},
				FormattedMessage: fmt.Sprintf("Moving volume %v\n", volume.Name),
			},
		)
	}

	srcDrive.ResetFinalizers()
	_, err = client.Drive().Update(
		ctx, srcDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		return fmt.Errorf("unable to remove volume references in source drive; %w", err)
	}
	return nil
}
