// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022, 2023 MinIO, Inc.
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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

var suspendVolumesCmd = &cobra.Command{
	Use:           "volumes [VOLUME ...]",
	Short:         "Suspend volumes",
	Long:          "Suspend the volumes (CAUTION: This will make the corresponding volumes as read-only)",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Suspend all volumes from a node
   $ kubectl {PLUGIN_NAME} suspend volumes --nodes=node1

2. Suspend specific volume from specific node
   $ kubectl {PLUGIN_NAME} suspend volumes --nodes=node1 --volumes=sda

3. Suspend a volume by its name 'pvc-0700b8c7-85b2-4894-b83a-274484f220d0'
   $ kubectl {PLUGIN_NAME} suspend volumes pvc-0700b8c7-85b2-4894-b83a-274484f220d0`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args

		if err := validateSuspendVolumesCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		if !dangerousFlag {
			utils.Eprintf(quietFlag, true, "Suspending the volumes will make them as read-only. Please review carefully before performing this *DANGEROUS* operation and retry this command with --dangerous flag.\n")
			os.Exit(1)
		}

		suspendVolumesMain(c.Context())
	},
}

func init() {
	setFlagOpts(suspendVolumesCmd)

	addNodesFlag(suspendVolumesCmd, "If present, suspend volumes from given nodes")
	addDrivesFlag(suspendVolumesCmd, "If present, suspend volumes by given drive names")
	addPodNameFlag(suspendVolumesCmd, "If present, suspend volumes by given pod names")
	addPodNSFlag(suspendVolumesCmd, "If present, suspend volumes by given pod namespaces")
	addDangerousFlag(suspendVolumesCmd, "Suspending the volumes will make them as read-only")
}

func validateSuspendVolumesCmd() error {
	if err := validateVolumeNameArgs(); err != nil {
		return err
	}
	if err := validateNodeArgs(); err != nil {
		return err
	}
	if err := validateDriveNameArgs(); err != nil {
		return err
	}
	if err := validatePodNameArgs(); err != nil {
		return err
	}
	if err := validatePodNSArgs(); err != nil {
		return err
	}

	switch {
	case len(volumeNameArgs) != 0:
	case len(nodesArgs) != 0:
	case len(drivesArgs) != 0:
	case len(podNameArgs) != 0:
	case len(podNSArgs) != 0:
	default:
		return errors.New("no volume selected to suspend")
	}

	return nil
}

func suspendVolumesMain(ctx context.Context) {
	var processed bool

	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		PodNameSelector(toLabelValues(podNameArgs)).
		PodNSSelector(toLabelValues(podNSArgs)).
		VolumeNameSelector(volumeNameArgs).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		processed = true

		if result.Volume.IsSuspended() {
			continue
		}

		volumeClient := client.VolumeClient()
		updateFunc := func() error {
			volume, err := volumeClient.Get(ctx, result.Volume.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			volume.Suspend()
			if !dryRunFlag {
				if _, err := volumeClient.Update(ctx, volume, metav1.UpdateOptions{}); err != nil {
					return err
				}
			}
			return nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, updateFunc); err != nil {
			utils.Eprintf(quietFlag, true, "unable to suspend volume %v; %v\n", result.Volume.Name, err)
			os.Exit(1)
		}

		if !quietFlag {
			fmt.Printf("Volume %v/%v suspended\n", result.Volume.GetNodeID(), result.Volume.Name)
		}
	}

	if !processed {
		utils.Eprintf(quietFlag, false, "No matching resources found\n")
		os.Exit(1)
	}
}
