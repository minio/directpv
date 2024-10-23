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
	"fmt"
	"os"
	"strings"
	"sync"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	devicepkg "github.com/minio/directpv/pkg/device"
	"github.com/minio/directpv/pkg/ellipsis"
	"github.com/minio/directpv/pkg/initrequest"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	drivesArgs []string
	initAll    bool
)

var bulkInitCmd = &cobra.Command{
	Use:           "bulk-init --drives [DRIVE-ELLIPSIS...]",
	Short:         "Bulk initialize the devices",
	SilenceUsage:  true,
	SilenceErrors: true,
	Hidden:        true,
	RunE: func(c *cobra.Command, _ []string) error {
		if err := sys.Mkdir(consts.MountRootDir, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		switch len(drivesArgs) {
		case 0:
			return errors.New("--drives must be provided")
		case 1:
			if drivesArgs[0] == "*" {
				initAll = true
			}
		}

		var drives []string
		for i := range drivesArgs {
			drivesArgs[i] = strings.TrimSpace(utils.TrimDevPrefix(drivesArgs[i]))
			if drivesArgs[i] == "" {
				return fmt.Errorf("empty drive name")
			}
			result, err := ellipsis.Expand(drivesArgs[i])
			if err != nil {
				return err
			}
			drives = append(drives, result...)
		}
		if !initAll && len(drives) == 0 {
			return errors.New("invalid ellipsis input; no drives selected")
		}
		return startBulkInit(c.Context(), drives)
	},
}

func init() {
	bulkInitCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", dryRunFlag, "No modify mode")
	bulkInitCmd.PersistentFlags().StringSliceVarP(&drivesArgs, "drives", "d", drivesArgs, "drives to be initialized; supports ellipses pattern e.g. sd{a...z}")
}

func startBulkInit(ctx context.Context, drives []string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	initRequestHandler, err := initrequest.NewHandler(
		ctx,
		nodeID,
		map[string]string{
			string(directpvtypes.TopologyDriverIdentity): identity,
			string(directpvtypes.TopologyDriverRack):     rack,
			string(directpvtypes.TopologyDriverZone):     zone,
			string(directpvtypes.TopologyDriverRegion):   region,
			string(directpvtypes.TopologyDriverNode):     string(nodeID),
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create initrequest handler; %v", err)
	}

	devices, err := devicepkg.Probe()
	if err != nil {
		return fmt.Errorf("unable to probe devices; %v", err)
	}

	var filteredDevices []devicepkg.Device
	for _, device := range devices {
		klog.Infoln(device.Name)
		if !device.Available() {
			continue
		}
		if initAll || utils.Contains(drives, device.Name) {
			filteredDevices = append(filteredDevices, device)
		}
	}

	if len(filteredDevices) == 0 {
		return errors.New("no available drives selected to initialize")
	}

	var wg sync.WaitGroup
	var failed bool
	for i := range filteredDevices {
		wg.Add(1)
		go func(device devicepkg.Device, force bool) {
			defer wg.Done()
			if dryRunFlag {
				klog.Infof("\n[DRY-RUN] initializing device %v with force: %v", device.Name, force)
				return
			}
			if err := initRequestHandler.InitDevice(device, force); err != nil {
				failed = true
				klog.ErrorS(err, "unable to init device %v", device.Name)
			}
		}(filteredDevices[i], filteredDevices[i].FSType() != "")
	}
	wg.Wait()

	if failed {
		return errors.New("failed to initialize all the drives")
	}

	return nil
}
