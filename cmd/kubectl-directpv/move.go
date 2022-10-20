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
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var moveCmd = &cobra.Command{
	Use:     "move SRC-DRIVE DEST-DRIVE",
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
			eprintf(quietFlag, true, "only one source and one destination drive must be provided\n")
			os.Exit(-1)
		}

		src := strings.TrimSpace(args[0])
		if src == "" {
			eprintf(quietFlag, true, "empty source drive\n")
			os.Exit(-1)
		}

		dest := strings.TrimSpace(args[1])
		if dest == "" {
			eprintf(quietFlag, true, "empty destination drive\n")
			os.Exit(-1)
		}

		moveMain(c.Context(), src, dest)
	},
}

func moveMain(ctx context.Context, src, dest string) {
	if src == dest {
		eprintf(quietFlag, true, "source and destination drives are same\n")
		os.Exit(1)
	}

	srcDrive, err := client.DriveClient().Get(ctx, src, metav1.GetOptions{})
	if err != nil {
		eprintf(quietFlag, true, "unable to get source drive; %v\n", err)
		os.Exit(1)
	}

	if !srcDrive.IsUnschedulable() {
		eprintf(quietFlag, true, "source drive is not cordoned\n")
		os.Exit(1)
	}

	var requiredCapacity int64
	var volumes []types.Volume
	for result := range volume.NewLister().VolumeNameSelector(srcDrive.GetVolumes()).List(ctx) {
		if result.Err != nil {
			eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		if result.Volume.IsPublished() {
			eprintf(quietFlag, true, "cannot move published volume %v\n", result.Volume.Name)
			os.Exit(1)
		}

		requiredCapacity += result.Volume.Status.TotalCapacity
		volumes = append(volumes, result.Volume)
	}

	if len(volumes) == 0 {
		eprintf(quietFlag, false, "No volumes found in source drive %v\n", src)
		return
	}

	destDrive, err := client.DriveClient().Get(ctx, dest, metav1.GetOptions{})
	if err != nil {
		eprintf(quietFlag, true, "unable to get destination drive; %v\n", err)
		os.Exit(1)
	}

	if destDrive.GetNodeID() != srcDrive.GetNodeID() {
		eprintf(
			quietFlag,
			true,
			"source and destination drives must be in same node; source node %v; desination node %v\n",
			srcDrive.GetNodeID(),
			destDrive.GetNodeID(),
		)
		os.Exit(1)
	}

	if !destDrive.IsUnschedulable() {
		eprintf(quietFlag, true, "destination drive is not cordoned\n")
		os.Exit(1)
	}

	if destDrive.Status.Status != directpvtypes.DriveStatusReady {
		eprintf(quietFlag, true, "destination drive is not ready state\n")
		os.Exit(1)
	}

	if srcDrive.GetAccessTier() != destDrive.GetAccessTier() {
		eprintf(
			quietFlag,
			true,
			"source drive access-tier %v and destination drive access-tier %v differ\n",
			srcDrive.GetAccessTier(),
			destDrive.GetAccessTier(),
		)
		os.Exit(1)
	}

	if destDrive.Status.FreeCapacity < requiredCapacity {
		eprintf(
			quietFlag,
			true,
			"insufficient free capacity on destination drive; required=%v free=%v\n",
			destDrive.Name,
			printableBytes(requiredCapacity),
			printableBytes(destDrive.Status.FreeCapacity),
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
		eprintf(quietFlag, true, "unable to move volumes to destination drive; %v\n", err)
		os.Exit(1)
	}

	for _, volume := range volumes {
		if !quietFlag {
			fmt.Println("Moving volume", volume.Name)
		}
	}

	srcDrive.ResetFinalizers()
	_, err = client.DriveClient().Update(
		ctx, srcDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		eprintf(quietFlag, true, "unable to remove volume references in source drive; %v\n", err)
		os.Exit(1)
	}
}
