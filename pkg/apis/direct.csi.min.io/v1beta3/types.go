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

package v1beta3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DirectCSIVolumeFinalizerPVProtection denotes PV protection finalizer.
	DirectCSIVolumeFinalizerPVProtection = Group + "/pv-protection"

	// DirectCSIVolumeFinalizerPurgeProtection denotes purge protection finalizer.
	DirectCSIVolumeFinalizerPurgeProtection = Group + "/purge-protection"

	// DirectCSIDriveFinalizerDataProtection denotes data protection finalizer.
	DirectCSIDriveFinalizerDataProtection = Group + "/data-protection"

	// DirectCSIDriveFinalizerPrefix denotes prefix finalizer.
	DirectCSIDriveFinalizerPrefix = Group + ".volume/"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectCSIDrive denotes drive CRD object.
type DirectCSIDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   DirectCSIDriveSpec   `json:"spec"`
	Status DirectCSIDriveStatus `json:"status,omitempty"`
}

// DirectCSIDriveSpec denotes drive specification.
type DirectCSIDriveSpec struct {
	// +optional
	RequestedFormat *RequestedFormat `json:"requestedFormat,omitempty"`
	// required
	DirectCSIOwned bool `json:"directCSIOwned"`
	// +optional
	DriveTaint map[string]string `json:"driveTaint,omitempty"`
}

// AccessTier denotes access tier.
type AccessTier string

const (
	// AccessTierWarm denotes "Warm" access tier.
	AccessTierWarm AccessTier = "Warm"

	// AccessTierHot denotes "Hot" access tier.
	AccessTierHot AccessTier = "Hot"

	// AccessTierCold denotes "Cold" access tier.
	AccessTierCold AccessTier = "Cold"

	// AccessTierUnknown denotes "Unknown" access tier.
	AccessTierUnknown AccessTier = "Unknown"
)

// DirectCSIDriveStatus denotes drive information.
type DirectCSIDriveStatus struct {
	Path string `json:"path"`
	// +optional
	AllocatedCapacity int64 `json:"allocatedCapacity,omitempty"`
	// +optional
	FreeCapacity int64 `json:"freeCapacity,omitempty"`
	// +optional
	RootPartition string `json:"rootPartition,omitempty"`
	// +optional
	PartitionNum int `json:"partitionNum,omitempty"`
	// +optional
	Filesystem string `json:"filesystem,omitempty"`
	// +optional
	Mountpoint string `json:"mountpoint,omitempty"`
	// +listType=atomic
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`
	// +optional
	NodeName string `json:"nodeName"`
	// +optional
	DriveStatus DriveStatus `json:"driveStatus,omitempty"`
	// +optional
	ModelNumber string `json:"modelNumber,omitempty"`
	// +optional
	SerialNumber string `json:"serialNumber,omitempty"`
	// +optional
	TotalCapacity int64 `json:"totalCapacity,omitempty"`
	// +optional
	PhysicalBlockSize int64 `json:"physicalBlockSize,omitempty"`
	// +optional
	LogicalBlockSize int64 `json:"logicalBlockSize,omitempty"`
	// +optional
	Topology map[string]string `json:"topology,omitempty"`
	// +optional
	AccessTier AccessTier `json:"accessTier,omitempty"`
	// +optional
	FilesystemUUID string `json:"filesystemUUID,omitempty"`
	// +optional
	PartitionUUID string `json:"partitionUUID,omitempty"`
	// +optional
	MajorNumber uint32 `json:"majorNumber,omitempty"`
	// +optional
	MinorNumber uint32 `json:"minorNumber,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	UeventSerial string `json:"ueventSerial,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	UeventFSUUID string `json:"ueventFSUUID,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	WWID string `json:"wwid,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	Vendor string `json:"vendor,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	DMName string `json:"dmName,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	DMUUID string `json:"dmUUID,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	MDUUID string `json:"mdUUID,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	PartTableUUID string `json:"partTableUUID,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	PartTableType string `json:"partTableType,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	Virtual bool `json:"virtual,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	ReadOnly bool `json:"readOnly,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	Partitioned bool `json:"partitioned,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	SwapOn bool `json:"swapOn,omitempty"`
	// +optional
	// +k8s:conversion-gen=false
	Master string `json:"master,omitempty"`
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// DirectCSIDriveCondition denotes drive condition.
type DirectCSIDriveCondition string

const (
	// DirectCSIDriveConditionOwned denotes "Owned" drive condition.
	DirectCSIDriveConditionOwned DirectCSIDriveCondition = "Owned"

	// DirectCSIDriveConditionMounted denotes "Mounted" drive condition.
	DirectCSIDriveConditionMounted DirectCSIDriveCondition = "Mounted"

	// DirectCSIDriveConditionFormatted denotes "Formatted" drive condition.
	DirectCSIDriveConditionFormatted DirectCSIDriveCondition = "Formatted"

	// DirectCSIDriveConditionInitialized denotes "Initialized" drive condition.
	DirectCSIDriveConditionInitialized DirectCSIDriveCondition = "Initialized"

	// DirectCSIDriveConditionReady denotes "NotReady" drive condition.
	DirectCSIDriveConditionReady DirectCSIDriveCondition = "Ready"
)

// DirectCSIDriveReason denotes drive reason.
type DirectCSIDriveReason string

const (
	// DirectCSIDriveReasonNotAdded denotes "NotAdded" drive reason.
	DirectCSIDriveReasonNotAdded DirectCSIDriveReason = "NotAdded"

	// DirectCSIDriveReasonAdded denotes "Added" drive reason.
	DirectCSIDriveReasonAdded DirectCSIDriveReason = "Added"

	// DirectCSIDriveReasonInitialized denotes "Initialized" drive reason.
	DirectCSIDriveReasonInitialized DirectCSIDriveReason = "Initialized"

	// DirectCSIDriveReasonNotReady denotes "NotReady" drive reason.
	DirectCSIDriveReasonNotReady DirectCSIDriveReason = "NotReady"

	// DirectCSIDriveReasonReady denotes "Ready" drive reason.
	DirectCSIDriveReasonReady DirectCSIDriveReason = "Ready"
)

// DirectCSIDriveMessage denotes drive message.
type DirectCSIDriveMessage string

const (
	// DirectCSIDriveMessageMounted denotes "Mounted" drive message.
	DirectCSIDriveMessageMounted DirectCSIDriveMessage = "Mounted"

	// DirectCSIDriveMessageNotMounted denotes "NotMounted" drive message.
	DirectCSIDriveMessageNotMounted DirectCSIDriveMessage = "NotMounted"

	// DirectCSIDriveMessageFormatted denotes "Formatted" drive message.
	DirectCSIDriveMessageFormatted DirectCSIDriveMessage = "Formatted"

	// DirectCSIDriveMessageNotFormatted denotes "NotFormatted" drive message.
	DirectCSIDriveMessageNotFormatted DirectCSIDriveMessage = "NotFormatted"
)

// RequestedFormat denotes drive format request information.
type RequestedFormat struct {
	// +optional
	Force bool `json:"force,omitempty"`
	// +optional
	Purge bool `json:"purge,omitempty"`
	// +optional
	Filesystem string `json:"filesystem,omitempty"`
	// +optional
	Mountpoint string `json:"mountpoint,omitempty"`
	// +listType=atomic
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`
}

// DriveStatus denotes drive status.
type DriveStatus string

const (
	// DriveStatusInUse denotes "InUse" drive status.
	DriveStatusInUse DriveStatus = "InUse"

	// DriveStatusAvailable denotes "Available" drive status.
	DriveStatusAvailable DriveStatus = "Available"

	// DriveStatusUnavailable denotes "Unavailable" drive status.
	DriveStatusUnavailable DriveStatus = "Unavailable"

	// DriveStatusReady denotes "Ready" drive status.
	DriveStatusReady DriveStatus = "Ready"

	// DriveStatusTerminating denotes "Terminating" drive status.
	DriveStatusTerminating DriveStatus = "Terminating"

	// DriveStatusReleased denotes "Released" drive status.
	DriveStatusReleased DriveStatus = "Released"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectCSIDriveList denotes list of drives.
type DirectCSIDriveList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectCSIDrive `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectCSIVolumeList denotes list of volumes.
type DirectCSIVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectCSIVolume `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectCSIVolume denotes volume CRD object.
type DirectCSIVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status DirectCSIVolumeStatus `json:"status,omitempty"`
}

// DirectCSIVolumeCondition denotes volume condition.
type DirectCSIVolumeCondition string

const (
	// DirectCSIVolumeConditionPublished denotes "Published" volume condition.
	DirectCSIVolumeConditionPublished DirectCSIVolumeCondition = "Published"

	// DirectCSIVolumeConditionStaged denotes "Staged" volume condition.
	DirectCSIVolumeConditionStaged DirectCSIVolumeCondition = "Staged"

	// DirectCSIVolumeConditionReady denotes "Ready" volume condition.
	DirectCSIVolumeConditionReady DirectCSIVolumeCondition = "Ready"
)

// DirectCSIVolumeReason denotes volume reason.
type DirectCSIVolumeReason string

const (
	// DirectCSIVolumeReasonNotInUse denotes "NotInUse" volume reason.
	DirectCSIVolumeReasonNotInUse DirectCSIVolumeReason = "NotInUse"

	// DirectCSIVolumeReasonInUse denotes "InUse" volume reason.
	DirectCSIVolumeReasonInUse DirectCSIVolumeReason = "InUse"

	// DirectCSIVolumeReasonReady denotes "Ready" volume reason.
	DirectCSIVolumeReasonReady DirectCSIVolumeReason = "Ready"

	// DirectCSIVolumeReasonNotReady denotes "NotReady" volume reason.
	DirectCSIVolumeReasonNotReady DirectCSIVolumeReason = "NotReady"
)

// DirectCSIVolumeStatus denotes volume information.
type DirectCSIVolumeStatus struct {
	// +optional
	Drive string `json:"drive,omitempty"`
	// +optional
	NodeName string `json:"nodeName,omitempty"`
	// +optional
	HostPath string `json:"hostPath,omitempty"`
	// +optional
	StagingPath string `json:"stagingPath,omitempty"`
	// +optional
	ContainerPath string `json:"containerPath,omitempty"`
	// +optional
	TotalCapacity int64 `json:"totalCapacity"`
	// +optional
	AvailableCapacity int64 `json:"availableCapacity"`
	// +optional
	UsedCapacity int64 `json:"usedCapacity"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}
