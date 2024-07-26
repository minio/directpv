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

package main

import (
	"context"
	"errors"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	drivepkg "github.com/minio/directpv/pkg/drive"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	forceFlag           = false
	disablePrefetchFlag = false
	dryRunFlag          = false
)

var repairCmd = &cobra.Command{
	Use:           "repair <DRIVE-ID>",
	Short:         "Start drive repair.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return errors.New("DRIVE-ID must be provided")
		case 1:
		default:
			return errors.New("only one DRIVE-ID must be provided")
		}
		return startRepair(c.Context(), args[0])
	},
}

func init() {
	repairCmd.PersistentFlags().BoolVar(&forceFlag, "force", forceFlag, "Force log zeroing")
	repairCmd.PersistentFlags().BoolVar(&disablePrefetchFlag, "disable-prefetch", disablePrefetchFlag, "Disable prefetching of inode and directory blocks")
	repairCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", dryRunFlag, "No modify mode")
}

func startRepair(ctx context.Context, driveID string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	drive, err := client.DriveClient().Get(ctx, driveID, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if drive.Status.Status != directpvtypes.DriveStatusRepairing {
		drive.Status.Status = directpvtypes.DriveStatusRepairing
	}

	updatedDrive, err := client.DriveClient().Update(ctx, drive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()})
	if err != nil {
		return err
	}

	return drivepkg.Repair(ctx, updatedDrive, forceFlag, disablePrefetchFlag, dryRunFlag)
}
