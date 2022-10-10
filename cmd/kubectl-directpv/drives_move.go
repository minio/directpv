// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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
	"fmt"
	"strings"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

var (
	fromDrive string
	toDrive   string

	errFromFlagNotSet       = errors.New("'--from' flag is not set")
	errToFlagNotSet         = errors.New("'--to' flag is not set")
	errInvalidState         = errors.New("invalid state")
	errCapacityNotAvailable = errors.New("the drive do not have enough capacity to accomodate")
)

var moveDriveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move a drive to another drive within the same node (without data)",
	Example: strings.ReplaceAll(
		`# Move a drive to another drive
$ kubectl {PLUGIN_NAME} drives move --from <drive_id> --to <drive_id>`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	RunE: func(c *cobra.Command, _ []string) error {
		if fromDrive == "" {
			return errFromFlagNotSet
		}
		if toDrive == "" {
			return errToFlagNotSet
		}
		return moveDrive(c.Context())
	},
}

func init() {
	moveDriveCmd.PersistentFlags().StringVarP(&fromDrive, "from", "", fromDrive, fmt.Sprintf("the name of the source %s drive that needs to be moved", consts.AppPrettyName))
	moveDriveCmd.PersistentFlags().StringVarP(&toDrive, "to", "", toDrive, fmt.Sprintf("the name of the target %s drive to which the source has to be moved", consts.AppPrettyName))
}

func moveDrive(ctx context.Context) error {
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		driveClient := client.DriveClient()
		sourceDrive, err := driveClient.Get(ctx, strings.TrimSpace(fromDrive), metav1.GetOptions{})
		if err != nil {
			return err
		}
		targetDrive, err := driveClient.Get(ctx, strings.TrimSpace(toDrive), metav1.GetOptions{})
		if err != nil {
			return err
		}
		if err := validateMoveRequest(sourceDrive, targetDrive); err != nil {
			return err
		}
		if err := move(ctx, sourceDrive, targetDrive); err != nil {
			return err
		}
		targetDrive.Status.Status = directpvtypes.DriveStatusMoving
		_, err = driveClient.Update(
			ctx, targetDrive, metav1.UpdateOptions{TypeMeta: types.NewDriveTypeMeta()},
		)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func validateMoveRequest(sourceDrive, targetDrive *types.Drive) error {
	if !(sourceDrive.IsCordoned() || sourceDrive.IsLost()) || !targetDrive.IsCordoned() {
		klog.Error("please make sure both the source and target drives are cordoned")
		return errInvalidState
	}
	if sourceDrive.Status.NodeName != targetDrive.Status.NodeName {
		klog.Error("both the source and target drives should be from same node")
		return errInvalidState
	}
	if sourceDrive.Status.AccessTier != targetDrive.Status.AccessTier {
		klog.Error("the source and target drives does not belong to the same access-tier")
		return errInvalidState
	}
	return nil
}

func move(ctx context.Context, sourceDrive, targetDrive *types.Drive) error {
	selector, err := getDriveNameSelectors([]string{sourceDrive.Name})
	if err != nil {
		return err
	}
	volumes, err := volume.GetVolumeList(ctx, nil, nil, nil, nil, selector)
	if err != nil {
		return err
	}
	for _, volume := range volumes {
		if volume.Status.DriveName != sourceDrive.Name {
			klog.Infof("invalid drive name %s found in volume %s", volume.Status.DriveName, volume.Name)
			return errInvalidState
		}
		if volume.Status.NodeName != sourceDrive.Status.NodeName {
			klog.Infof("invalid node name %s found in volume %s", volume.Status.NodeName, volume.Name)
			return errInvalidState
		}
		if volume.IsPublished() {
			klog.Info("please make sure all the volumes in the source drive are not inuse")
			return fmt.Errorf("volume %s is still published", volume.Name)
		}
		if targetDrive.Status.FreeCapacity < volume.Status.TotalCapacity {
			klog.Info("the target drive cannot accomodate the volumes from the source")
			return errCapacityNotAvailable
		}
		finalizer := consts.DriveFinalizerPrefix + volume.Name
		if !utils.ItemIn(targetDrive.Finalizers, finalizer) {
			targetDrive.Status.FreeCapacity -= volume.Status.TotalCapacity
			targetDrive.Status.AllocatedCapacity += volume.Status.TotalCapacity
			targetDrive.SetFinalizers(append(targetDrive.GetFinalizers(), finalizer))
		}
	}
	return nil
}
