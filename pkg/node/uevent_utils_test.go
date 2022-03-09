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

package node

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta3"
	"github.com/minio/directpv/pkg/client"
	clientsetfake "github.com/minio/directpv/pkg/clientset/fake"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/utils"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestSetDriveStatus(t *testing.T) {
	testCases := []struct {
		device        *sys.Device
		drive         *directcsi.DirectCSIDrive
		expectedDrive *directcsi.DirectCSIDrive
	}{
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				Parent:            "parent",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
		},
		// fill missing hwinfo
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				Parent:            "parent",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					RootPartition:     "sdb1",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					DMName:            "vg0-lv0",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
		},
		// make unavailabe drive to available if the conditions look fine
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				Parent:            "parent",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/data",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusUnavailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
		},
		// make available drive as unavailable
		{
			device: &sys.Device{
				Name:              "sdb1",
				Major:             8,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				Parent:            "parent",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/data"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/data",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/data",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusUnavailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
		},
		// path change
		{
			device: &sys.Device{
				Name:              "sdc1",
				Major:             9,
				Minor:             1,
				Size:              uint64(512000),
				WWID:              "ABCD000000001234567",
				Model:             "QEMU",
				Serial:            "1A2B3C4D",
				Vendor:            "KVM",
				DMName:            "vg0-lv0",
				DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
				MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
				PTUUID:            "7e3bf265-0396-440b-88fd-dc2003505583",
				PTType:            "gpt",
				PartUUID:          "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventSerial:      "12345ABCD678",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				Parent:            "parent",
				FreeCapacity:      uint64(412000),
				LogicalBlockSize:  uint64(512),
				PhysicalBlockSize: uint64(512),
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Partition:         1,
			},
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdb1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdb1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdb1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       8,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
			expectedDrive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
					Labels: map[string]string{
						"direct.csi.min.io/access-tier": "Unknown",
						"direct.csi.min.io/created-by":  "directcsi-driver",
						"direct.csi.min.io/node":        "test-node",
						"direct.csi.min.io/path":        "sdc1",
						"direct.csi.min.io/version":     "v1beta3",
					},
				},
				Status: directcsi.DirectCSIDriveStatus{
					NodeName:          "test-node",
					Path:              "/dev/sdc1",
					Filesystem:        "xfs",
					TotalCapacity:     512000,
					FreeCapacity:      412000,
					AllocatedCapacity: 100000,
					LogicalBlockSize:  512,
					ModelNumber:       "QEMU",
					Mountpoint:        "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:      []string{"relatime", "rw"},
					PartitionNum:      1,
					PhysicalBlockSize: 512,
					RootPartition:     "sdc1",
					SerialNumber:      "1A2B3C4D",
					FilesystemUUID:    "d79dff9e-2884-46f2-8919-dada2eecb12d",
					PartitionUUID:     "d895e5a6-c5cb-49d7-ae0d-a3946f4f1a3a",
					MajorNumber:       9,
					MinorNumber:       1,
					UeventSerial:      "12345ABCD678",
					UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
					WWID:              "ABCD000000001234567",
					Vendor:            "KVM",
					DMName:            "vg0-lv0",
					DMUUID:            "0361196e-e683-46cf-9f38-711ee498af05",
					MDUUID:            "1f5ecc9b-de46-43fe-89d6-bf58ee85b155",
					PartTableUUID:     "7e3bf265-0396-440b-88fd-dc2003505583",
					PartTableType:     "gpt",
					DriveStatus:       directcsi.DriveStatusAvailable,
					Topology: map[string]string{
						string(utils.TopologyDriverIdentity): "identity",
						string(utils.TopologyDriverRack):     "rack",
						string(utils.TopologyDriverZone):     "zone",
						string(utils.TopologyDriverRegion):   "region",
						string(utils.TopologyDriverNode):     "test-node",
					},
					Conditions: []metav1.Condition{
						{
							Type:   string(directcsi.DirectCSIDriveConditionOwned),
							Status: metav1.ConditionTrue,
							Reason: string(directcsi.DirectCSIDriveReasonAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionMounted),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionFormatted),
							Status:  metav1.ConditionTrue,
							Message: "xfs",
							Reason:  string(directcsi.DirectCSIDriveReasonNotAdded),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionInitialized),
							Status:  metav1.ConditionTrue,
							Message: "initialized",
							Reason:  string(directcsi.DirectCSIDriveReasonInitialized),
						},
						{
							Type:    string(directcsi.DirectCSIDriveConditionReady),
							Status:  metav1.ConditionTrue,
							Message: "",
							Reason:  string(directcsi.DirectCSIDriveReasonReady),
						},
					},
				},
			},
		},
	}

	handler := createDriveEventHandler()
	for i, testCase := range testCases {
		updatedDrive := handler.setDriveStatus(testCase.device, testCase.drive)
		if !reflect.DeepEqual(updatedDrive, testCase.expectedDrive) {
			t.Fatalf("case %v: expected %v, got: %v", i, testCase.expectedDrive, updatedDrive)
		}
	}
}

func TestValidateDrive(t *testing.T) {
	testCases := []struct {
		drive        *directcsi.DirectCSIDrive
		device       *sys.Device
		expectedErrs []error
	}{
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					DriveStatus: directcsi.DriveStatusAvailable,
				},
			},
			device: nil,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					DriveStatus: directcsi.DriveStatusUnavailable,
				},
			},
			device: nil,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					DriveStatus: directcsi.DriveStatusReleased,
				},
			},
			device: nil,
		},
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					DriveStatus: directcsi.DriveStatusTerminating,
				},
			},
			device: nil,
		},
		// no error
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
		},
		// Mountpoint mismatch
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/data"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/data",
			},
			expectedErrs: []error{
				errInvalidMount,
				errInvalidDrive(
					"Mountpoint",
					"/var/lib/direct-csi/mnt/fsuuid",
					"/data",
				),
			},
		},
		// Mountpoint options mismatch
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/data"},
				FirstMountOptions: []string{"relatime", "rw"},
				FirstMountPoint:   "/data",
			},
			expectedErrs: []error{
				errInvalidMount,
				errInvalidDrive(
					"Mountpoint",
					"/var/lib/direct-csi/mnt/fsuuid",
					"/data",
				),
				errInvalidDrive(
					"MountpointOptions",
					[]string{"prjquota", "relatime", "rw"},
					[]string{"relatime", "rw"},
				),
			},
		},
		// FSUUID mismatch
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "sdfgqf9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "sdfgqf9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
			expectedErrs: []error{
				errInvalidDrive(
					"UeventFSUUID",
					"d79dff9e-2884-46f2-8919-dada2eecb12d",
					"sdfgqf9e-2884-46f2-8919-dada2eecb12d"),
				errInvalidDrive(
					"FilesystemUUID",
					"d79dff9e-2884-46f2-8919-dada2eecb12d",
					"sdfgqf9e-2884-46f2-8919-dada2eecb12d"),
			},
		},
		// Filesystem mismatch
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "ext4",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
			expectedErrs: []error{
				errInvalidDrive(
					"Filesystem",
					"xfs",
					"ext4"),
			},
		},
		// size mismatch
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              16777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
			},
			expectedErrs: []error{
				fmt.Errorf(
					"the size of the drive is less than %v",
					sys.MinSupportedDeviceSize),
			},
		},
		// ReadOnly
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				ReadOnly:          true,
			},
			expectedErrs: []error{
				errInvalidDrive(
					"ReadOnly",
					false,
					true),
			},
		},
		// SwapOn
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				SwapOn:            true,
			},
			expectedErrs: []error{
				errInvalidDrive(
					"SwapOn",
					false,
					true),
			},
		},
		// master
		{
			drive: &directcsi.DirectCSIDrive{
				TypeMeta: utils.DirectCSIDriveTypeMeta(),
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-drive",
				},
				Status: directcsi.DirectCSIDriveStatus{
					Path:           "/dev/sdc1",
					DriveStatus:    directcsi.DriveStatusReady,
					Filesystem:     "xfs",
					Mountpoint:     "/var/lib/direct-csi/mnt/fsuuid",
					MountOptions:   []string{"prjquota", "relatime", "rw"},
					FilesystemUUID: "d79dff9e-2884-46f2-8919-dada2eecb12d",
					UeventFSUUID:   "d79dff9e-2884-46f2-8919-dada2eecb12d",
				},
			},
			device: &sys.Device{
				Size:              36777215,
				Name:              "sdc1",
				FSUUID:            "d79dff9e-2884-46f2-8919-dada2eecb12d",
				FSType:            "xfs",
				UeventFSUUID:      "d79dff9e-2884-46f2-8919-dada2eecb12d",
				MountPoints:       []string{"/var/lib/direct-csi/mnt/fsuuid"},
				FirstMountOptions: []string{"prjquota", "relatime", "rw"},
				FirstMountPoint:   "/var/lib/direct-csi/mnt/fsuuid",
				Master:            "vda",
			},
			expectedErrs: []error{
				errInvalidDrive(
					"Master",
					"",
					"vda"),
			},
		},
	}

	for i, testCase := range testCases {
		err := validateDrive(testCase.drive, testCase.device)
		errs := multierr.Errors(err)
		if !reflect.DeepEqual(errs, testCase.expectedErrs) {
			t.Fatalf("case %v: expected errs: %v got %v", i, testCase.expectedErrs, errs)
		}
	}
}

func TestSyncVolumeLabels(t *testing.T) {
	testDriveObjects := []runtime.Object{
		&directcsi.DirectCSIDrive{
			TypeMeta: utils.DirectCSIDriveTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-drive-1",
				Finalizers: []string{
					string(directcsi.DirectCSIDriveFinalizerDataProtection),
					directcsi.DirectCSIDriveFinalizerPrefix + "test-volume-1",
					directcsi.DirectCSIDriveFinalizerPrefix + "test-volume-2",
				},
			},
			Status: directcsi.DirectCSIDriveStatus{
				Path:        "/dev/sdb1",
				NodeName:    "test-node",
				DriveStatus: directcsi.DriveStatusInUse,
			},
		},
	}
	testVolumeObjects := []runtime.Object{
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-1",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
				Labels: map[string]string{
					"direct.csi.min.io/drive-path": "sda1",
				},
			},
		},
		&directcsi.DirectCSIVolume{
			TypeMeta: utils.DirectCSIVolumeTypeMeta(),
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-volume-2",
				Finalizers: []string{
					string(directcsi.DirectCSIVolumeFinalizerPurgeProtection),
				},
				Labels: map[string]string{
					"direct.csi.min.io/drive-path": "sda1",
				},
			},
		},
	}

	ctx := context.TODO()
	client.SetLatestDirectCSIDriveInterface(clientsetfake.NewSimpleClientset(testDriveObjects...).DirectV1beta3().DirectCSIDrives())
	client.SetLatestDirectCSIVolumeInterface(clientsetfake.NewSimpleClientset(testVolumeObjects...).DirectV1beta3().DirectCSIVolumes())

	if err := syncVolumeLabels(ctx, testDriveObjects[0].(*directcsi.DirectCSIDrive)); err != nil {
		t.Fatalf("could not set volume labels: %v", err)
	}

	result, err := client.GetLatestDirectCSIVolumeInterface().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("could not set volume labels: %v", err)
	}

	for _, item := range result.Items {
		labels := item.GetLabels()
		if labels == nil {
			t.Error("empty labels found")
		}
		value, ok := labels[string(utils.DrivePathLabelKey)]
		if !ok {
			t.Fatalf("no label value found by the key: %s", string(utils.DrivePathLabelKey))
		}
		if value != "sdb1" {
			t.Fatalf("drive path label mismatch. expected: sdb1 got %s", value)
		}
	}

}
