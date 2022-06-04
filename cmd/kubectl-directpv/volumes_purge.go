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
	"fmt"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/spf13/cobra"
)

var purgeVolumesCmd = &cobra.Command{
	Use:   "purge",
	Short: utils.BinaryNameTransform("purge released|failed volumes in the {{ . }} cluster. This command has to be cautiously used as it may lead to data loss."),
	Long:  "",
	Example: utils.BinaryNameTransform(`
# Purge all released|failed volumes in the cluster
$ kubectl {{ . }} volumes purge --all

# Purge the volume by its name(id)
$ kubectl {{ . }} volumes purge <volume-name>

# Purge all released|failed volumes from a particular node
$ kubectl {{ . }} volumes purge --nodes=direct-1

# Combine multiple filters using csv
$ kubectl {{ . }} volumes purge --nodes=direct-1,direct-2 --drives=/dev/nvme0n1

# Purge all released|failed volumes by pod name
$ kubectl {{ . }} volumes purge --pod-name=minio-{1...3}

# Purge all released|failed volumes by pod namespace
$ kubectl {{ . }} volumes purge --pod-namespace=tenant-{1...3}

# Purge all released|failed volumes based on drive and volume ellipses
$ kubectl {{ . }} volumes purge --drives '/dev/xvd{a...d} --nodes 'node-{1...4}''
`),
	RunE: func(c *cobra.Command, args []string) error {
		if !all {
			if len(drives) == 0 && len(nodes) == 0 && len(podNames) == 0 && len(podNss) == 0 && len(args) == 0 {
				return fmt.Errorf("atleast one of '%s', '%s', '%s', '%s' or '%s' must be specified",
					utils.Bold("--all"),
					utils.Bold("--drives"),
					utils.Bold("--nodes"),
					utils.Bold("--pod-name"),
					utils.Bold("--pod-namespace"),
				)
			}
		}
		if err := validateVolumeSelectors(); err != nil {
			return err
		}
		if len(driveGlobs) > 0 || len(nodeGlobs) > 0 || len(podNameGlobs) > 0 || len(podNsGlobs) > 0 {
			klog.Warning("Glob matches will be deprecated soon. Please use ellipses instead")
		}
		return purgeVolumes(c.Context(), args)
	},
}

func init() {
	purgeVolumesCmd.PersistentFlags().StringSliceVarP(&drives, "drives", "d", drives, "filter by drive path(s) (also accepts ellipses range notations)")
	purgeVolumesCmd.PersistentFlags().StringSliceVarP(&nodes, "nodes", "n", nodes, "filter by node name(s) (also accepts ellipses range notations)")
	purgeVolumesCmd.PersistentFlags().BoolVarP(&all, "all", "a", all, "purge all released|failed volumes")
	purgeVolumesCmd.PersistentFlags().StringSliceVarP(&podNames, "pod-name", "", podNames, "filter by pod name(s) (also accepts ellipses range notations)")
	purgeVolumesCmd.PersistentFlags().StringSliceVarP(&podNss, "pod-namespace", "", podNss, "filter by pod namespace(s) (also accepts ellipses range notations)")
}

func purgeVolumes(ctx context.Context, IDArgs []string) error {
	return processFilteredVolumes(
		ctx,
		IDArgs,
		func(volume *directcsi.DirectCSIVolume) bool {
			pv, err := client.GetKubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					return true
				}
				klog.Infof("couldn't fetch the pv %s due to %s", volume.Name, err.Error())
				return false
			}
			switch pv.Status.Phase {
			case corev1.VolumeReleased, corev1.VolumeFailed:
				return true
			default:
				klog.Infof("couldn't purge the volume %s as the pv is in %s phase", volume.Name, string(pv.Status.Phase))
				return false
			}
		},
		func(volume *directcsi.DirectCSIVolume) error {
			finalizers := volume.GetFinalizers()
			updatedFinalizers := []string{}
			for _, f := range finalizers {
				if f == directcsi.DirectCSIVolumeFinalizerPVProtection {
					continue
				}
				updatedFinalizers = append(updatedFinalizers, f)
			}
			volume.SetFinalizers(updatedFinalizers)
			return nil
		},
		func(ctx context.Context, volume *directcsi.DirectCSIVolume) error {
			if _, err := client.GetLatestDirectCSIVolumeInterface().Update(ctx, volume, metav1.UpdateOptions{
				TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			}); err != nil {
				return err
			}
			if err := client.GetLatestDirectCSIVolumeInterface().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil {
				if !k8serrors.IsNotFound(err) {
					return err
				}
			}
			return nil
		},
		VolumePurge,
	)
}
