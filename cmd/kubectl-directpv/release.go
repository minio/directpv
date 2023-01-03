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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/utils"
	"github.com/minio/directpv/pkg/volume"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var releaseCmd = &cobra.Command{
	Use:           "release [VOLUME ...]",
	Short:         "Cleanup stale volumes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`# Release all stale volumes
$ kubectl {PLUGIN_NAME} release --all

# Release a volume by its ID
$ kubectl {PLUGIN_NAME} release pvc-6355041d-f9c6-4bd6-9335-f2bccbe73929

# Release volumes served by drive name in all nodes.
$ kubectl {PLUGIN_NAME} release --drives=nvme1n1

# Release volumes served by drive
$ kubectl {PLUGIN_NAME} release --drive-id=78e6486e-22d2-4c93-99d0-00f4e3a8411f

# Release volumes served by a node
$ kubectl {PLUGIN_NAME} release --nodes=node1

# Release volumes by pod name
$ kubectl {PLUGIN_NAME} release --pod-names=minio-{1...3}

# Release volumes by pod namespace
$ kubectl {PLUGIN_NAME} release --pod-namespaces=tenant-{1...3}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args

		if err := validateReleaseCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		releaseMain(c.Context())
	},
}

func init() {
	releaseCmd.Flags().SortFlags = false
	releaseCmd.InheritedFlags().SortFlags = false
	releaseCmd.LocalFlags().SortFlags = false
	releaseCmd.LocalNonPersistentFlags().SortFlags = false
	releaseCmd.NonInheritedFlags().SortFlags = false
	releaseCmd.PersistentFlags().SortFlags = false

	addNodesFlag(releaseCmd, "If present, select volumes from given nodes")
	addDrivesFlag(releaseCmd, "If present, select volumes by given drive names")
	addAllFlag(releaseCmd, "If present, select all volumes")
	addDryRunFlag(releaseCmd)
	addDriveIDFlag(releaseCmd, "Select volumes by drive IDs")
	addPodNameFlag(releaseCmd, "Select volumes by pod names")
	addPodNSFlag(releaseCmd, "Select volumes by pod namespaces")
}

func validateReleaseCmd() error {
	if err := validateNodeArgs(); err != nil {
		return err
	}

	if err := validateDriveNameArgs(); err != nil {
		return err
	}

	if err := validateDriveIDArgs(); err != nil {
		return err
	}

	if err := validatePodNameArgs(); err != nil {
		return err
	}

	if err := validatePodNSArgs(); err != nil {
		return err
	}

	if err := validateVolumeNameArgs(); err != nil {
		return err
	}

	switch {
	case allFlag:
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(driveIDArgs) != 0:
	case len(podNameArgs) != 0:
	case len(podNSArgs) != 0:
	case len(volumeNameArgs) != 0:
	default:
		return errors.New("no volume selected to release")
	}

	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveIDArgs = nil
		podNameArgs = nil
		podNSArgs = nil
		volumeNameArgs = nil
	}

	return nil
}

func releaseMain(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := volume.NewLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		DriveIDSelector(toLabelValues(driveIDArgs)).
		PodNameSelector(toLabelValues(podNameArgs)).
		PodNSSelector(toLabelValues(podNSArgs)).
		StatusSelector(volumeStatusSelectors).
		VolumeNameSelector(volumeNameArgs).
		List(ctx)

	matchFunc := func(volume *types.Volume) bool {
		pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true
			}
			utils.Eprintf(quietFlag, true, "unable to get PV for volume %v; %v\n", volume.Name, err)
			return false
		}
		switch pv.Status.Phase {
		case corev1.VolumeReleased, corev1.VolumeFailed:
			return true
		default:
			if !quietFlag {
				utils.Eprintf(quietFlag, false, "Skipping PV %v in %v state to volume %v\n", pv.Name, pv.Status.Phase, volume.Name)
			}
			return false
		}
	}

	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		if !matchFunc(&result.Volume) {
			continue
		}

		result.Volume.RemovePVProtection()

		if dryRunFlag {
			continue
		}

		if _, err := client.VolumeClient().Update(ctx, &result.Volume, metav1.UpdateOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		}); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(1)
		}
		if err := client.VolumeClient().Delete(ctx, result.Volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(1)
		}

		if !quietFlag {
			fmt.Println("Removing volume", result.Volume.Name)
		}
	}
}
