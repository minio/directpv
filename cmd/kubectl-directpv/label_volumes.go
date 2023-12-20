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
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

var labelVolumesCmd = &cobra.Command{
	Use:           "volumes k=v|k-",
	Aliases:       []string{"volume", "vol"},
	Short:         "Set labels to volumes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Set 'tier: hot' label to all volumes in all nodes
   $ kubectl {PLUGIN_NAME} label volumes tier=hot --all

2. Set 'type: fast' to volumes allocated in specific drives from a node
   $ kubectl {PLUGIN_NAME} label volumes type=fast --nodes=node1 --drives=nvme1n{1...3}

3. Remove 'tier: hot' label from all volumes in all nodes
   $ kubectl {PLUGIN_NAME} label volumes tier- --all`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = idArgs
		if err := validateLabelVolumesCmd(args); err != nil {
			utils.Eprintf(quietFlag, true, "%s; Check `--help` for usage\n", err.Error())
			os.Exit(1)
		}
		labelVolumesMain(c.Context())
	},
}

func validateLabelVolumesCmd(args []string) (err error) {
	if err = validateLabelArgs(); err != nil {
		return err
	}
	if err = validateListVolumesArgs(); err != nil {
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
	case len(volumeStatusArgs) != 0:
	case len(labelArgs) != 0:
	default:
		return errors.New("no volumes selected to label")
	}
	if allFlag {
		nodesArgs = nil
		drivesArgs = nil
		driveIDArgs = nil
		podNameArgs = nil
		podNSArgs = nil
		volumeNameArgs = nil
		volumeStatusArgs = nil
		labelArgs = nil
	}
	labels, err = validateLabelCmdArgs(args)
	return
}

func init() {
	setFlagOpts(labelVolumesCmd)

	addDriveIDFlag(labelVolumesCmd, "Filter output by drive IDs")
	addPodNameFlag(labelVolumesCmd, "Filter output by pod names")
	addPodNSFlag(labelVolumesCmd, "Filter output by pod namespaces")
	addVolumeStatusFlag(labelVolumesCmd, "Filter output by volume status")
	addLabelsFlag(labelVolumesCmd, "If present, select by volume labels")
	addIDFlag(labelVolumesCmd, "If present, select by volume ID")
}

func labelVolumesMain(ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	resultCh := client.NewVolumeLister().
		NodeSelector(toLabelValues(nodesArgs)).
		DriveNameSelector(toLabelValues(drivesArgs)).
		DriveIDSelector(toLabelValues(driveIDArgs)).
		PodNameSelector(toLabelValues(podNameArgs)).
		PodNSSelector(toLabelValues(podNSArgs)).
		StatusSelector(volumeStatusSelectors).
		VolumeNameSelector(volumeNameArgs).
		LabelSelector(labelSelectors).
		List(ctx)
	for result := range resultCh {
		if result.Err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", result.Err)
			os.Exit(1)
		}

		var verb string
		volume := &result.Volume
		for i := range labels {
			updateFunc := func() (err error) {
				if labels[i].remove {
					if ok := volume.RemoveLabel(labels[i].key); !ok {
						return
					}
					verb = "removed from"
				} else {
					if ok := volume.SetLabel(labels[i].key, labels[i].value); !ok {
						return
					}
					verb = "set on"
				}
				if !dryRunFlag {
					volume, err = client.VolumeClient().Update(ctx, volume, metav1.UpdateOptions{})
				}
				if err != nil {
					utils.Eprintf(quietFlag, true, "%v: %v\n", volume.Name, err)
				} else if !quietFlag {
					fmt.Printf("Label '%s' successfully %s %v\n", labels[i].String(), verb, volume.Name)
				}
				return
			}
			retry.RetryOnConflict(retry.DefaultRetry, updateFunc)
		}
	}
}
