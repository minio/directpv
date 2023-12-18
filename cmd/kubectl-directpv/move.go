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

	"github.com/fatih/color"
	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/jobs"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

var (
	withData, overwrite bool
	moveCmd             = &cobra.Command{
		Use:           "move SRC-DRIVE DEST-DRIVE",
		Aliases:       []string{"mv"},
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Move volumes excluding data from source drive to destination drive on a same node",
		Example: strings.ReplaceAll(
			`1. Move volumes from drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 to drive 834e8f4c-14f4-49b9-9b77-e8ac854108d5
   $ kubectl {PLUGIN_NAME} drives move af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 834e8f4c-14f4-49b9-9b77-e8ac854108d5

2. Move volumes from drive af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 to drive 834e8f4c-14f4-49b9-9b77-e8ac854108d5 with data
$ kubectl {PLUGIN_NAME} drives move af3b8b4c-73b4-4a74-84b7-1ec30492a6f0 834e8f4c-14f4-49b9-9b77-e8ac854108d5 --with-data`,
			`{PLUGIN_NAME}`,
			consts.AppName,
		),
		Run: func(c *cobra.Command, args []string) {
			if len(args) != 2 {
				utils.Eprintf(quietFlag, true, "only one source and one destination drive must be provided\n")
				os.Exit(-1)
			}

			src := strings.TrimSpace(args[0])
			if src == "" {
				utils.Eprintf(quietFlag, true, "empty source drive\n")
				os.Exit(-1)
			}

			dest := strings.TrimSpace(args[1])
			if dest == "" {
				utils.Eprintf(quietFlag, true, "empty destination drive\n")
				os.Exit(-1)
			}

			if overwrite && !withData {
				utils.Eprintf(quietFlag, true, "'--overwrite' flag must be set only when '--with-data' flag is set")
			}

			moveMain(c.Context(), src, dest)
		},
	}
)

func init() {
	moveCmd.PersistentFlags().BoolVar(&withData, "with-data", withData, "move the volume with data")
	moveCmd.PersistentFlags().BoolVar(&overwrite, "overwrite", overwrite, "overwrite any duplicate volume copy job if present")
}

func moveMain(ctx context.Context, src, dest string) {
	if src == dest {
		utils.Eprintf(quietFlag, true, "source and destination drives are same\n")
		os.Exit(1)
	}

	srcDrive, err := client.DriveClient().Get(ctx, src, metav1.GetOptions{})
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to get source drive; %v\n", err)
		os.Exit(1)
	}

	if !srcDrive.IsUnschedulable() {
		utils.Eprintf(quietFlag, true, "source drive is not cordoned\n")
		os.Exit(1)
	}

	sourceVolumeNames := srcDrive.GetVolumes()
	if len(sourceVolumeNames) == 0 {
		utils.Eprintf(quietFlag, false, "No volumes found in source drive %v\n", src)
		return
	}

	var requiredCapacity int64
	var volumes []types.Volume
	for result := range volume.NewLister().VolumeNameSelector(sourceVolumeNames).List(ctx) {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		if result.Volume.IsPublished() {
			utils.Eprintf(quietFlag, true, "cannot move published volume %v\n", result.Volume.Name)
			os.Exit(1)
		}

		requiredCapacity += result.Volume.Status.TotalCapacity
		volumes = append(volumes, result.Volume)
	}

	if len(volumes) == 0 {
		utils.Eprintf(quietFlag, false, "No volumes found in source drive %v\n", src)
		return
	}

	destDrive, err := client.DriveClient().Get(ctx, dest, metav1.GetOptions{})
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to get destination drive; %v\n", err)
		os.Exit(1)
	}

	if destDrive.GetNodeID() != srcDrive.GetNodeID() {
		utils.Eprintf(
			quietFlag,
			true,
			"source and destination drives must be in same node; source node %v; desination node %v\n",
			srcDrive.GetNodeID(),
			destDrive.GetNodeID(),
		)
		os.Exit(1)
	}

	if !destDrive.IsUnschedulable() {
		utils.Eprintf(quietFlag, true, "destination drive is not cordoned\n")
		os.Exit(1)
	}

	switch {
	case destDrive.Status.Status == directpvtypes.DriveStatusReady:
	case destDrive.Status.Status == directpvtypes.DriveStatusMoving && withData:
	default:
		utils.Eprintf(quietFlag, true, "destination drive is not in ready state\n")
		os.Exit(1)
	}

	if srcDrive.GetAccessTier() != destDrive.GetAccessTier() {
		utils.Eprintf(
			quietFlag,
			true,
			"source drive access-tier %v and destination drive access-tier %v differ\n",
			srcDrive.GetAccessTier(),
			destDrive.GetAccessTier(),
		)
		os.Exit(1)
	}

	if destDrive.Status.FreeCapacity < requiredCapacity {
		utils.Eprintf(
			quietFlag,
			true,
			"insufficient free capacity on destination drive; required=%v free=%v\n",
			printableBytes(requiredCapacity),
			printableBytes(destDrive.Status.FreeCapacity),
		)
		os.Exit(1)
	}

	var jobParams jobs.ContainerParams
	if withData {
		jobParams.Image, jobParams.ImagePullSecrets, jobParams.Tolerations, err = getContainerParams(ctx)
		if err != nil {
			utils.Eprintf(quietFlag, true, "unable to get container params; %v", err)
			os.Exit(1)
		}
		srcDrive.AddCopyProtectionFinalizer()
	}

	var processed bool
	for _, volume := range volumes {
		if withData {
			if err := jobs.CreateCopyJob(ctx, jobs.CopyOpts{
				SourceDriveID:      srcDrive.GetDriveID(),
				DestinationDriveID: destDrive.GetDriveID(),
				VolumeID:           volume.Name,
				NodeID:             srcDrive.GetNodeID(),
			}, jobParams, overwrite); err != nil {
				if apierrors.IsAlreadyExists(err) && !overwrite {
					utils.Eprintf(quietFlag, false, "duplicate job found for %v; Please use `--overwrite` for this volume to be moved", volume.Name)
				}
				utils.Eprintf(quietFlag, true, "unable to create copy job for volume %v; %v", volume.Name, err)
				continue
			}
			if err := setVolumeStatusCopying(ctx, volume.Name); err != nil {
				utils.Eprintf(quietFlag, true, "unable to set volume status moving; %v", err)
				os.Exit(1)
			}
		}
		processed = true
		if !quietFlag {
			fmt.Println("Moving volume ", volume.Name)
		}
		if destDrive.AddVolumeFinalizer(volume.Name) {
			destDrive.Status.FreeCapacity -= volume.Status.TotalCapacity
			destDrive.Status.AllocatedCapacity += volume.Status.TotalCapacity
		}
		srcDrive.RemoveVolumeFinalizer(volume.Name)
	}

	if !processed {
		os.Exit(1)
	}

	destDrive.Status.Status = directpvtypes.DriveStatusMoving
	_, err = client.DriveClient().Update(
		ctx, destDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to move volumes to destination drive; %v\n", err)
		os.Exit(1)
	}

	_, err = client.DriveClient().Update(
		ctx, srcDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
	)
	if err != nil {
		utils.Eprintf(quietFlag, true, "unable to remove volume references in source drive; %v\n", err)
		os.Exit(1)
	}

	if withData && !quietFlag {
		color.HiGreen("Jobs created successfully to copy the volume data. Please uncordon the node to get the jobs scheduled")
	}
}

func setVolumeStatusCopying(ctx context.Context, volumeName string) error {
	updateFunc := func() error {
		volume, err := client.VolumeClient().Get(ctx, volumeName, metav1.GetOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		})
		if err != nil {
			return err
		}
		volume.Status.Status = directpvtypes.VolumeStatusCopying
		_, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		})
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
}
