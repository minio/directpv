// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package discovery

import (
	"context"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/utils"

	"github.com/google/uuid"
)

const (
	loopBackDeviceCount = 4
)

// NewDiscovery creates drive discovery.
func NewDiscovery(ctx context.Context, identity, nodeID, rack, zone, region string) (*Discovery, error) {
	config, err := utils.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	topologies := map[string]string{}
	topologies[utils.TopologyDriverIdentity] = identity
	topologies[utils.TopologyDriverRack] = rack
	topologies[utils.TopologyDriverZone] = zone
	topologies[utils.TopologyDriverRegion] = region
	topologies[utils.TopologyDriverNode] = nodeID

	directClientset, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	d := &Discovery{
		NodeID:          nodeID,
		directcsiClient: directClientset,
		driveTopology:   topologies,
	}

	if err := d.readRemoteDrives(ctx); err != nil {
		return nil, err
	}

	if err := d.readMounts(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Discovery) readMounts() (err error) {
	d.mounts, err = sys.ProbeMounts()
	return
}

func (d *Discovery) readRemoteDrives(ctx context.Context) error {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	nodeLabelValue, err := utils.NewLabelValue(d.NodeID)
	if err != nil {
		return err
	}

	resultCh, err := utils.ListDrives(ctx,
		d.directcsiClient.DirectV1beta3().DirectCSIDrives(),
		[]utils.LabelValue{nodeLabelValue},
		nil,
		nil,
		utils.MaxThreadCount,
	)
	if err != nil {
		return err
	}

	var remoteDriveList []*remoteDrive
	for result := range resultCh {
		if result.Err != nil {
			return result.Err
		}
		// Assign decoded drive path in case this drive information converted from v1alpha1/v1beta1
		result.Drive.Status.Path = utils.GetDrivePath(&result.Drive)
		remoteDriveList = append(remoteDriveList, &remoteDrive{DirectCSIDrive: result.Drive})
	}
	d.remoteDrives = remoteDriveList
	return nil
}

// Init initializes drive discovery.
func (d *Discovery) Init(ctx context.Context, loopBackOnly bool) error {
	localDrives, err := d.findLocalDrives(ctx, loopBackOnly)
	if err != nil {
		return err
	}

	localDriveStates := d.toDirectCSIDriveStatus(localDrives)
	var unidentifedDriveStates []directcsi.DirectCSIDriveStatus
	if len(d.remoteDrives) == 0 {
		for _, localDriveState := range localDriveStates {
			if err := d.createNewDrive(ctx, localDriveState); err != nil {
				return err
			}
		}
	} else {
		for _, localDriveState := range localDriveStates {
			remoteDrive, err := d.identify(localDriveState)
			if err == nil {
				if err := d.syncRemoteDrive(ctx, localDriveState, remoteDrive); err != nil {
					return err
				}
				continue
			}
			unidentifedDriveStates = append(unidentifedDriveStates, localDriveState)
		}

		for _, localDriveState := range unidentifedDriveStates {
			remoteDrive, isNotSynced, err := d.identifyDriveByLegacyName(localDriveState)
			if err == nil {
				if isNotSynced {
					if err := d.syncRemoteDrive(ctx, localDriveState, remoteDrive); err != nil {
						return err
					}
					continue
				}
			}
			if err := d.createNewDrive(ctx, localDriveState); err != nil {
				return err
			}
		}
	}

	// Delete the unmapped remote drives
	if err := d.deleteUnmatchedRemoteDrives(ctx); err != nil {
		return err
	}

	return nil
}

func (d *Discovery) createNewDrive(ctx context.Context, localDriveState directcsi.DirectCSIDriveStatus) error {
	return utils.CreateDrive(
		ctx,
		d.directcsiClient.DirectV1beta3().DirectCSIDrives(),
		utils.NewDirectCSIDrive(uuid.New().String(), localDriveState),
	)
}

func (d *Discovery) syncRemoteDrive(ctx context.Context, localDriveState directcsi.DirectCSIDriveStatus, remoteDrive *remoteDrive) error {
	return d.syncDrive(ctx, utils.NewDirectCSIDrive(remoteDrive.Name, localDriveState))
}

func (d *Discovery) findLocalDrives(ctx context.Context, loopBackOnly bool) (map[string]*sys.Device, error) {
	if loopBackOnly {
		// Flush the existing loopback setups
		if err := sys.FlushLoopBackReservations(); err != nil {
			return nil, err
		}
		// Reserve loopbacks
		if err := sys.ReserveLoopbackDevices(loopBackDeviceCount); err != nil {
			return nil, err
		}
	}

	devices, err := sys.ProbeDevices()
	if err != nil {
		return nil, err
	}

	for name := range devices {
		if (sys.LoopRegexp.MatchString(name) && !loopBackOnly) || devices[name].Size == 0 {
			delete(devices, name)
		}
	}

	return devices, nil
}

func (d *Discovery) toDirectCSIDriveStatus(devices map[string]*sys.Device) []directcsi.DirectCSIDriveStatus {
	statusList := []directcsi.DirectCSIDriveStatus{}
	for _, device := range devices {
		statusList = append(statusList, utils.NewDirectCSIDriveStatus(device, d.NodeID, d.driveTopology))
	}
	return statusList
}
