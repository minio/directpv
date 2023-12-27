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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
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
	if err := admin.LabelVolumes(ctx, admin.LabelVolumeArgs{
		Nodes:          nodesArgs,
		Drives:         drivesArgs,
		DriveIDs:       driveIDArgs,
		PodNames:       podNameArgs,
		PodNamespaces:  podNSArgs,
		VolumeStatus:   volumeStatusSelectors,
		VolumeNames:    volumeNameArgs,
		LabelSelectors: labelSelectors,
	}, labels); err != nil {
		utils.Eprintf(quietFlag, !errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
