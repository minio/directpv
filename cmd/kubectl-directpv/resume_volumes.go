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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var resumeVolumesCmd = &cobra.Command{
	Use:           "volumes [VOLUME ...]",
	Short:         "Resume suspended volumes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Resume all volumes from a node
   $ kubectl {PLUGIN_NAME} resume volumes --nodes=node1

2. Resume specific volume from specific node
   $ kubectl {PLUGIN_NAME} resume volumes --nodes=node1 --volumes=sda

3. Resume a volume by its name 'pvc-0700b8c7-85b2-4894-b83a-274484f220d0'
   $ kubectl {PLUGIN_NAME} resume volumes pvc-0700b8c7-85b2-4894-b83a-274484f220d0`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args

		if err := validateResumeVolumesCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		resumeVolumesMain(c.Context())
	},
}

func init() {
	setFlagOpts(resumeVolumesCmd)

	addNodesFlag(resumeVolumesCmd, "If present, resume volumes from given nodes")
	addDrivesFlag(resumeVolumesCmd, "If present, resume volumes by given drive names")
	addPodNameFlag(resumeVolumesCmd, "If present, resume volumes by given pod names")
	addPodNSFlag(resumeVolumesCmd, "If present, resume volumes by given pod namespaces")
}

func validateResumeVolumesCmd() error {
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
		return errors.New("no volume selected to resume")
	}

	return nil
}

func resumeVolumesMain(ctx context.Context) {
	if err := admin.ResumeVolumes(ctx, admin.ResumeVolumeArgs{
		Nodes:         nodesArgs,
		Drives:        drivesArgs,
		PodNames:      podNameArgs,
		PodNamespaces: podNSArgs,
		VolumeNames:   volumeNameArgs,
		Quiet:         quietFlag,
		DryRun:        dryRunFlag,
	}); err != nil {
		utils.Eprintf(quietFlag, !errors.Is(err, admin.ErrNoMatchingResourcesFound), "%v\n", err)
		os.Exit(1)
	}
}
