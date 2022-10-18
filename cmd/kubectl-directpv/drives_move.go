// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var drivesMoveCmd = &cobra.Command{
	Use:     "move <src-drive-id> <dest-drive-id>",
	Aliases: []string{"mv"},
	Short:   "Move volumes excluding data from source drive to destination drive on a same node.",
	Example: strings.ReplaceAll(
		`# Move volumes on drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 to drive 834e8f4c-14f4-49b9-9b77-e8ac854108d5
$ kubectl {PLUGIN_NAME} drives move af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 834e8f4c-14f4-49b9-9b77-e8ac854108d5`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if len(args) != 2 {
			eprintf("only one source and one destination drive must be provided", true)
			os.Exit(-1)
		}

		src := strings.TrimSpace(args[0])
		if src == "" {
			eprintf("empty source drive", true)
			os.Exit(-1)
		}

		dest := strings.TrimSpace(args[1])
		if dest == "" {
			eprintf("empty destination drive", true)
			os.Exit(-1)
		}

		drivesMoveMain(c.Context(), src, dest)
	},
}

func drivesMoveMain(ctx context.Context, src, dest string) {
	if src == dest {
		eprintf("source and destination drives are same", true)
		os.Exit(1)
	}

	srcDrive, err := client.DriveClient().Get(ctx, src, metav1.GetOptions{})
	if err != nil {
		eprintf(fmt.Sprintf("unable to get source drive; %v", err), true)
		os.Exit(1)
	}

	if !srcDrive.IsUnschedulable() {
		eprintf("source drive is not cordoned", true)
		os.Exit(1)
	}

	var requiredCapacity int64
	var volumes []types.Volume
	for result := range getVolumesByNames(ctx, srcDrive.GetVolumes(), true) {
		if result.Err != nil {
			eprintf(result.Err.Error(), true)
			os.Exit(1)
		}

		if result.Volume.IsPublished() {
			eprintf(fmt.Sprintf("cannot move published volume %v", result.Volume.Name), true)
			os.Exit(1)
		}

		requiredCapacity += result.Volume.Status.TotalCapacity
		volumes = append(volumes, result.Volume)
	}

	if len(volumes) == 0 {
		eprintf(fmt.Sprintf("No volumes found in source drive %v", src), false)
		return
	}

	destDrive, err := client.DriveClient().Get(ctx, dest, metav1.GetOptions{})
	if err != nil {
		eprintf(fmt.Sprintf("unable to get destination drive; %v", err), true)
		os.Exit(1)
	}

	if destDrive.GetNodeID() != srcDrive.GetNodeID() {
		eprintf(
			fmt.Sprintf(
				"source and destination drives must be in same node; source node %v; desination node %v",
				srcDrive.GetNodeID(),
				destDrive.GetNodeID(),
			),
			true,
		)
		os.Exit(1)
	}

	if !destDrive.IsUnschedulable() {
		eprintf("destination drive is not cordoned", true)
		os.Exit(1)
	}

	if destDrive.Status.Status != directpvtypes.DriveStatusReady {
		eprintf("destination drive is not ready state", true)
		os.Exit(1)
	}

	if srcDrive.GetAccessTier() != destDrive.GetAccessTier() {
		eprintf(
			fmt.Sprintf(
				"source drive access-tier %v and destination drive access-tier %v are different",
				srcDrive.GetAccessTier(),
				destDrive.GetAccessTier(),
			),
			true,
		)
		os.Exit(1)
	}

	if destDrive.Status.FreeCapacity < requiredCapacity {
		eprintf(
			fmt.Sprintf(
				"insufficient free capacity on destination drive; required=%v free=%v",
				destDrive.Name,
				printableBytes(requiredCapacity),
				printableBytes(destDrive.Status.FreeCapacity),
			),
			true,
		)
		os.Exit(1)
	}

	for _, volume := range volumes {
		if destDrive.AddVolumeFinalizer(volume.Name) {
			destDrive.Status.FreeCapacity -= requiredCapacity
			destDrive.Status.AllocatedCapacity += requiredCapacity
		}
	}
	destDrive.Status.Status = directpvtypes.DriveStatusMoving
	_, err = client.DriveClient().Update(
		ctx, destDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		eprintf(fmt.Sprintf("unable to move volumes to destination drive; %v", err), true)
		os.Exit(1)
	}

	for _, volume := range volumes {
		eprintf(fmt.Sprintf("Moving volume %v", volume.Name), false)
	}

	srcDrive.ResetFinalizers()
	_, err = client.DriveClient().Update(
		ctx, srcDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		eprintf(fmt.Sprintf("unable to remove volume references in source drive; %v", err), true)
		os.Exit(1)
	}
}
