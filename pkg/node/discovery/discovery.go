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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	simd "github.com/minio/sha256-simd"
)

const (
	loopBackDeviceCount = 4
)

var loopRegexp = regexp.MustCompile("loop[0-9].*")

var unknownDriveCounter int32

func isDirectCSIMount(mountPoints []string) bool {
	if len(mountPoints) == 0 {
		return true
	}

	for _, mountPoint := range mountPoints {
		if strings.HasPrefix(mountPoint, "/var/lib/direct-csi/") {
			return true
		}
	}
	return false
}

func toDirectCSIDriveStatus(nodeID string, topology map[string]string, device *sys.Device) directcsi.DirectCSIDriveStatus {
	driveStatus := directcsi.DriveStatusAvailable
	if device.ReadOnly || device.Partitioned || device.SwapOn || device.Master != "" || !isDirectCSIMount(device.MountPoints) {
		driveStatus = directcsi.DriveStatusUnavailable
	}

	mounted := metav1.ConditionFalse
	if device.FirstMountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	formatted := metav1.ConditionFalse
	if device.FSType != "" {
		formatted = metav1.ConditionTrue
	}

	return directcsi.DirectCSIDriveStatus{
		AccessTier:        directcsi.AccessTierUnknown,
		DriveStatus:       driveStatus,
		Filesystem:        device.FSType,
		FreeCapacity:      int64(device.FreeCapacity),
		AllocatedCapacity: int64(device.Size - device.FreeCapacity),
		LogicalBlockSize:  int64(device.LogicalBlockSize),
		ModelNumber:       device.Model,
		MountOptions:      device.FirstMountOptions,
		Mountpoint:        device.FirstMountPoint,
		NodeName:          nodeID,
		PartitionNum:      device.Partition,
		Path:              "/dev/" + device.Name,
		PhysicalBlockSize: int64(device.PhysicalBlockSize),
		RootPartition:     device.Name,
		SerialNumber:      device.Serial,
		TotalCapacity:     int64(device.Size),
		FilesystemUUID:    device.FSUUID,
		PartitionUUID:     device.PartUUID,
		MajorNumber:       uint32(device.Major),
		MinorNumber:       uint32(device.Minor),
		Topology:          topology,
		Conditions: []metav1.Condition{
			{
				Type:               string(directcsi.DirectCSIDriveConditionOwned),
				Status:             metav1.ConditionFalse,
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionMounted),
				Status:             mounted,
				Message:            device.FirstMountPoint,
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionFormatted),
				Status:             formatted,
				Message:            "xfs",
				Reason:             string(directcsi.DirectCSIDriveReasonNotAdded),
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               string(directcsi.DirectCSIDriveConditionInitialized),
				Status:             metav1.ConditionTrue,
				Message:            "",
				Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
				LastTransitionTime: metav1.Now(),
			},
		},
	}
}

func NewDiscovery(ctx context.Context, identity, nodeID, rack, zone, region string) (*Discovery, error) {
	config, err := utils.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	topologies := map[string]string{}
	topologies[topology.TopologyDriverIdentity] = identity
	topologies[topology.TopologyDriverRack] = rack
	topologies[topology.TopologyDriverZone] = zone
	topologies[topology.TopologyDriverRegion] = region
	topologies[topology.TopologyDriverNode] = nodeID

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
	resultCh, err := utils.ListDrives(ctx,
		d.directcsiClient.DirectV1beta2().DirectCSIDrives(),
		[]string{d.NodeID}, nil, nil, utils.MaxThreadCount,
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
			remoteDrive, err := d.Identify(localDriveState)
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
	directCSIClient := d.directcsiClient.DirectV1beta2()
	driveClient := directCSIClient.DirectCSIDrives()

	newDrive := makeDirectCSIDrive(localDriveState, "")
	if _, err := driveClient.Create(ctx, newDrive, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (d *Discovery) syncRemoteDrive(ctx context.Context, localDriveState directcsi.DirectCSIDriveStatus, remoteDrive *remoteDrive) error {
	identifiedLegacyDrive := makeDirectCSIDrive(localDriveState, remoteDrive.Name)
	if err := d.syncDrive(ctx, identifiedLegacyDrive); err != nil {
		return err
	}
	return nil
}

func makeDirectCSIDrive(driveStatus directcsi.DirectCSIDriveStatus, driveName string) *directcsi.DirectCSIDrive {
	if driveName == "" {
		driveName = uuid.New().String()
	}
	return &directcsi.DirectCSIDrive{
		ObjectMeta: metav1.ObjectMeta{
			Name: driveName,
			Labels: map[string]string{
				utils.NodeLabel:       driveStatus.NodeName,
				utils.DrivePathLabel:  filepath.Base(driveStatus.Path),
				utils.VersionLabel:    directcsi.Version,
				utils.CreatedByLabel:  "directcsi-driver",
				utils.AccessTierLabel: string(driveStatus.AccessTier),
			},
		},
		Status: driveStatus,
	}
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
		if (loopRegexp.MatchString(name) && !loopBackOnly) || devices[name].Size == 0 {
			delete(devices, name)
		}
	}

	return devices, nil
}

func (d *Discovery) toDirectCSIDriveStatus(devices map[string]*sys.Device) []directcsi.DirectCSIDriveStatus {
	statusList := []directcsi.DirectCSIDriveStatus{}
	for _, device := range devices {
		statusList = append(statusList, toDirectCSIDriveStatus(d.NodeID, d.driveTopology, device))
	}
	return statusList
}

func makeV1beta1DriveName(nodeID, path string) string {
	driveName := strings.Join([]string{nodeID, path}, "-")
	return fmt.Sprintf("%x", simd.Sum256([]byte(driveName)))
}
