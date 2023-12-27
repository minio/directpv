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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CleanArgs represents the arguments to clean the volumes
type CleanArgs struct {
	Nodes         []string
	Drives        []string
	DriveIDs      []string
	PodNames      []string
	PodNamespaces []string
	VolumeStatus  []directpvtypes.VolumeStatus
	VolumeNames   []string
	Quiet         bool
	DryRun        bool
}

// Clean removes the stale/abandoned volumes
func Clean(ctx context.Context, args CleanArgs) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		NodeSelector(utils.ToLabelValues(args.Nodes)).
		DriveNameSelector(utils.ToLabelValues(args.Drives)).
		DriveIDSelector(utils.ToLabelValues(args.DriveIDs)).
		PodNameSelector(utils.ToLabelValues(args.PodNames)).
		PodNSSelector(utils.ToLabelValues(args.PodNamespaces)).
		StatusSelector(args.VolumeStatus).
		VolumeNameSelector(args.VolumeNames).
		List(ctx)

	matchFunc := func(volume *types.Volume) bool {
		pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true
			}
			utils.Eprintf(args.Quiet, true, "unable to get PV for volume %v; %v\n", volume.Name, err)
			return false
		}
		switch pv.Status.Phase {
		case corev1.VolumeReleased, corev1.VolumeFailed:
			return true
		default:
			return false
		}
	}

	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}
		if !matchFunc(&result.Volume) {
			continue
		}
		result.Volume.RemovePVProtection()
		if args.DryRun {
			continue
		}
		if _, err := client.VolumeClient().Update(ctx, &result.Volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		}); err != nil {
			return err
		}
		if err := client.VolumeClient().Delete(ctx, result.Volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if !args.Quiet {
			fmt.Println("Removing volume", result.Volume.Name)
		}
	}

	return nil
}
