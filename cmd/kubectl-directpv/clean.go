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
	"os"
	"strings"

	"github.com/minio/directpv/pkg/admin"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:           "clean [VOLUME ...]",
	Short:         "Cleanup stale volumes",
	SilenceUsage:  true,
	SilenceErrors: true,
	Example: strings.ReplaceAll(
		`1. Cleanup all stale volumes
   $ kubectl {PLUGIN_NAME} clean --all

2. Clean a volume by its ID
   $ kubectl {PLUGIN_NAME} clean pvc-6355041d-f9c6-4bd6-9335-f2bccbe73929

3. Clean volumes served by drive name in all nodes.
   $ kubectl {PLUGIN_NAME} clean --drives=nvme1n1

4. Clean volumes served by drive
   $ kubectl {PLUGIN_NAME} clean --drive-id=78e6486e-22d2-4c93-99d0-00f4e3a8411f

5. Clean volumes served by a node
   $ kubectl {PLUGIN_NAME} clean --nodes=node1

6. Clean volumes by pod name
   $ kubectl {PLUGIN_NAME} clean --pod-names=minio-{1...3}

7. Clean volumes by pod namespace
   $ kubectl {PLUGIN_NAME} clean --pod-namespaces=tenant-{1...3}`,
		`{PLUGIN_NAME}`,
		consts.AppName,
	),
	Run: func(c *cobra.Command, args []string) {
		volumeNameArgs = args

		if err := validateCleanCmd(); err != nil {
			utils.Eprintf(quietFlag, true, "%v\n", err)
			os.Exit(-1)
		}

		cleanMain(c.Context())
	},
}

func init() {
	setFlagOpts(cleanCmd)

	addNodesFlag(cleanCmd, "If present, select volumes from given nodes")
	addDrivesFlag(cleanCmd, "If present, select volumes by given drive names")
	addAllFlag(cleanCmd, "If present, select all volumes")
	addDryRunFlag(cleanCmd, "Run in dry run mode")
	addDriveIDFlag(cleanCmd, "Select volumes by drive IDs")
	addPodNameFlag(cleanCmd, "Select volumes by pod names")
	addPodNSFlag(cleanCmd, "Select volumes by pod namespaces")
}

func validateCleanCmd() error {
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
		return errors.New("no volume selected to clean")
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

func cleanMain(ctx context.Context) {
	if err := admin.Clean(ctx, admin.CleanArgs{
		Nodes:         nodesArgs,
		Drives:        drivesArgs,
		DriveIDs:      driveIDArgs,
		PodNames:      podNameArgs,
		PodNamespaces: podNSArgs,
		VolumeStatus:  volumeStatusSelectors,
		VolumeNames:   volumeNameArgs,
	}); err != nil {
		utils.Eprintf(quietFlag, true, "%v\n", err)
		os.Exit(1)
	}
}
