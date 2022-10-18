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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	"github.com/minio/directpv/pkg/types"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var volumesPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge released and failed " + consts.AppName + "volumes. CAUTION: THIS MAY LEAD TO DATA LOSS",
	Example: strings.ReplaceAll(
		`# Purge all released|failed volumes
$ kubectl {PLUGIN_NAME} volumes purge --all

# Purge the volume by its name(id)
$ kubectl {PLUGIN_NAME} volumes purge <volume-name>

# Purge all released|failed volumes from a particular node
$ kubectl {PLUGIN_NAME} volumes purge --nodes=node1

# Combine multiple filters using csv
$ kubectl {PLUGIN_NAME} volumes purge --nodes=node1,node2 --drives=/dev/nvme0n1

# Purge all released|failed volumes by pod name
$ kubectl {PLUGIN_NAME} volumes purge --pod-name=minio-{1...3}

# Purge all released|failed volumes by pod namespace
$ kubectl {PLUGIN_NAME} volumes purge --pod-namespace=tenant-{1...3}

# Purge all released|failed volumes based on drive and volume ellipses
$ kubectl {PLUGIN_NAME} volumes purge --drives /dev/xvd{a...d} --nodes node-{1...4}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		if !allFlag && len(driveArgs) == 0 && len(nodeArgs) == 0 && len(podNameArgs) == 0 && len(podNSArgs) == 0 && len(args) == 0 {
			eprintf("one of '--all', '--drives', '--nodes', '--pod-name' or '--pod-namespace' must be specified", true)
			os.Exit(-1)
		}

		volumesPurgeMain(c.Context(), args)
	},
}

func volumesPurgeMain(ctx context.Context, names []string) {
	err := processFilteredVolumes(
		ctx,
		names,
		func(volume *types.Volume) bool {
			pv, err := k8s.KubeClient().CoreV1().PersistentVolumes().Get(ctx, volume.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true
				}
				eprintf(fmt.Sprintf("unable to get PV for volume %v; %v", volume.Name, err), true)
				return false
			}
			switch pv.Status.Phase {
			case corev1.VolumeReleased, corev1.VolumeFailed:
				return true
			default:
				if !quiet {
					eprintf(fmt.Sprintf("Skipping volume %v as associated PV is in %v phase", volume.Name, pv.Status.Phase), false)
				}
				return false
			}
		},
		func(volume *types.Volume) error {
			volume.RemovePVProtection()
			return nil
		},
		func(ctx context.Context, volume *types.Volume) error {
			if _, err := client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{
				TypeMeta: types.NewVolumeTypeMeta(),
			}); err != nil {
				return err
			}
			if err := client.VolumeClient().Delete(ctx, volume.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			return nil
		},
		"drive-purge",
	)

	if err != nil {
		eprintf(err.Error(), true)
		os.Exit(1)
	}
}
