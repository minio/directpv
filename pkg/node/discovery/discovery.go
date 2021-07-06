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
	"strings"

	directcsi "github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1beta2"
	"github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/sys"
	"github.com/minio/direct-csi/pkg/sys/gpt"
	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	simd "github.com/minio/sha256-simd"
)

const (
	loopBackDeviceCount = 4
)

var unknownDriveCounter int32

func NewDiscovery(ctx context.Context, identity, nodeID, rack, zone, region string) (*Discovery, error) {
	kubeConfig := utils.GetKubeConfig()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
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

func (d *Discovery) readMounts() error {
	mounts, err := sys.ProbeMountInfo()
	if err != nil {
		return err
	}
	d.mounts = mounts
	return nil
}

func (d *Discovery) readRemoteDrives(ctx context.Context) error {
	directCSIClient := d.directcsiClient.DirectV1beta2()
	driveClient := directCSIClient.DirectCSIDrives()
	driveList, err := driveClient.List(ctx, metav1.ListOptions{
		TypeMeta: utils.DirectCSIDriveTypeMeta(),
	})
	if err != nil {
		return err
	}
	drives := driveList.Items

	var remoteDriveList []*remoteDrive
	for _, drive := range drives {
		if drive.Status.NodeName == d.NodeID {
			remoteDrive := &remoteDrive{
				matched:        false,
				DirectCSIDrive: drive,
			}
			remoteDriveList = append(remoteDriveList, remoteDrive)
		}
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
				directcsi.Group + "/node":        driveStatus.NodeName,
				directcsi.Group + "/path":        filepath.Base(driveStatus.Path),
				directcsi.Group + "/version":     directcsi.Version,
				directcsi.Group + "/created-by":  "directcsi-driver",
				directcsi.Group + "/access-tier": string(driveStatus.AccessTier),
			},
		},
		Status: driveStatus,
	}
}

func (d *Discovery) findLocalDrives(ctx context.Context, loopBackOnly bool) ([]sys.BlockDevice, error) {
	if loopBackOnly {
		// Flush the existing loopback setups
		if err := sys.FlushLoopBackReservations(); err != nil {
			return []sys.BlockDevice{}, err
		}
		// Reserve loopbacks
		if err := sys.ReserveLoopbackDevices(loopBackDeviceCount); err != nil {
			return []sys.BlockDevice{}, err
		}
	}

	devs, err := sys.FindDevices(ctx, loopBackOnly)
	if err != nil {
		return []sys.BlockDevice{}, err
	}

	return devs, nil
}

func (d *Discovery) toDirectCSIDriveStatus(localDrives []sys.BlockDevice) []directcsi.DirectCSIDriveStatus {
	driveStatusList := []directcsi.DirectCSIDriveStatus{}
	nodeID := d.NodeID
	for _, localDrive := range localDrives {
		partitions := localDrive.GetPartitions()
		if len(partitions) > 0 {
			for _, partition := range partitions {
				driveStatus := d.directCSIDriveStatusFromPartition(nodeID, partition, localDrive.Devname, localDrive.DeviceError)
				driveStatusList = append(driveStatusList, driveStatus)
			}
			continue
		}
		driveStatus := d.directCSIDriveStatusFromRoot(nodeID, localDrive)
		driveStatusList = append(driveStatusList, driveStatus)
	}
	return driveStatusList
}

func (d *Discovery) directCSIDriveStatusFromPartition(nodeID string, partition sys.Partition, rootPartition string, blockErr error) directcsi.DirectCSIDriveStatus {
	var fs, UUID string
	if partition.FSInfo != nil {
		fs = string(partition.FSInfo.FSType)
		UUID = string(partition.FSInfo.UUID)
	}

	var allocatedCapacity, freeCapacity, totalCapacity int64
	if partition.FSInfo != nil {
		freeCapacity = int64(partition.FSInfo.FreeCapacity)
		totalCapacity = int64(partition.FSInfo.TotalCapacity)
		allocatedCapacity = totalCapacity - freeCapacity
	}

	var mountOptions []string
	var mountPoint string
	var mounts []sys.MountInfo
	var driveStatus directcsi.DriveStatus

	driveStatus = directcsi.DriveStatusAvailable
	if partition.FSInfo != nil {
		mounts = partition.FSInfo.Mounts
		for _, m := range mounts {
			if m.Mountpoint == "/" {
				driveStatus = directcsi.DriveStatusUnavailable
			}
		}
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].Mountpoint
		}
	}
	_, ok := gpt.SystemPartitionTypes[partition.TypeUUID]
	if ok || blockErr != nil {
		driveStatus = directcsi.DriveStatusUnavailable
	}

	blockInitializationStatus := metav1.ConditionTrue
	if blockErr != nil {
		blockInitializationStatus = metav1.ConditionFalse
	}

	mounted := metav1.ConditionFalse
	formatted := metav1.ConditionFalse
	if fs != "" {
		formatted = metav1.ConditionTrue
	}
	if mountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	return directcsi.DirectCSIDriveStatus{
		AccessTier:        directcsi.AccessTierUnknown,
		DriveStatus:       driveStatus,
		Filesystem:        fs,
		FreeCapacity:      freeCapacity,
		AllocatedCapacity: allocatedCapacity,
		LogicalBlockSize:  int64(partition.LogicalBlockSize),
		ModelNumber:       "", // Fix Me
		MountOptions:      mountOptions,
		Mountpoint:        mountPoint,
		NodeName:          nodeID,
		PartitionNum:      int(partition.PartitionNum),
		Path:              partition.Path,
		PhysicalBlockSize: int64(partition.PhysicalBlockSize),
		RootPartition:     rootPartition,
		SerialNumber:      partition.SerialNumber,
		TotalCapacity:     totalCapacity,
		FilesystemUUID:    UUID,
		PartitionUUID:     partition.PartitionGUID,
		MajorNumber:       partition.Major,
		MinorNumber:       partition.Minor,
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
				Message:            mountPoint,
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
				Type:   string(directcsi.DirectCSIDriveConditionInitialized),
				Status: blockInitializationStatus,
				Message: func() string {
					if blockErr == nil {
						return ""
					}
					return blockErr.Error()
				}(),
				Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
				LastTransitionTime: metav1.Now(),
			},
		},
		Topology: d.driveTopology,
	}
}

func (d *Discovery) directCSIDriveStatusFromRoot(nodeID string, blockDevice sys.BlockDevice) directcsi.DirectCSIDriveStatus {
	var fs, UUID string
	if blockDevice.FSInfo != nil {
		fs = string(blockDevice.FSInfo.FSType)
		UUID = string(blockDevice.FSInfo.UUID)
	}

	var freeCapacity, totalCapacity, allocatedCapacity int64
	if blockDevice.FSInfo != nil {
		freeCapacity = int64(blockDevice.FSInfo.FreeCapacity)
		totalCapacity = int64(blockDevice.FSInfo.TotalCapacity)
		allocatedCapacity = totalCapacity - freeCapacity
	}

	var mountOptions []string
	var mountPoint string
	var mounts []sys.MountInfo
	var driveStatus directcsi.DriveStatus

	driveStatus = directcsi.DriveStatusAvailable
	if blockDevice.FSInfo != nil {
		mounts = blockDevice.FSInfo.Mounts
		for _, m := range mounts {
			if m.Mountpoint == "/" {
				driveStatus = directcsi.DriveStatusUnavailable
			}
		}
		if len(mounts) > 0 {
			mountOptions = mounts[0].MountFlags
			mountPoint = mounts[0].Mountpoint
		}
	}

	blockInitializationStatus := metav1.ConditionTrue
	if blockDevice.DeviceError != nil {
		driveStatus = directcsi.DriveStatusUnavailable
		blockInitializationStatus = metav1.ConditionFalse
	}

	mounted := metav1.ConditionFalse
	formatted := metav1.ConditionFalse
	if fs != "" {
		formatted = metav1.ConditionTrue
	}
	if mountPoint != "" {
		mounted = metav1.ConditionTrue
	}

	return directcsi.DirectCSIDriveStatus{
		AccessTier:        directcsi.AccessTierUnknown,
		DriveStatus:       driveStatus,
		Filesystem:        fs,
		FreeCapacity:      freeCapacity,
		AllocatedCapacity: allocatedCapacity,
		LogicalBlockSize:  int64(blockDevice.LogicalBlockSize),
		ModelNumber:       "", // Fix Me
		MountOptions:      mountOptions,
		Mountpoint:        mountPoint,
		NodeName:          nodeID,
		PartitionNum:      int(0),
		Path:              blockDevice.Path,
		PhysicalBlockSize: int64(blockDevice.PhysicalBlockSize),
		RootPartition:     blockDevice.Devname,
		SerialNumber:      blockDevice.SerialNumber,
		TotalCapacity:     totalCapacity,
		FilesystemUUID:    UUID,
		PartitionUUID:     "",
		MajorNumber:       blockDevice.Major,
		MinorNumber:       blockDevice.Minor,
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
				Message:            mountPoint,
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
				Status:             blockInitializationStatus,
				Message:            blockDevice.Error(),
				Reason:             string(directcsi.DirectCSIDriveReasonInitialized),
				LastTransitionTime: metav1.Now(),
			},
		},
		Topology: d.driveTopology,
	}
}

func makeV1beta1DriveName(nodeID, path string) string {
	driveName := strings.Join([]string{nodeID, path}, "-")
	return fmt.Sprintf("%x", simd.Sum256([]byte(driveName)))
}
