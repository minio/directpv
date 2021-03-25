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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DirectCSIVolumeFinalizerPVProtection    = Group + "/pv-protection"
	DirectCSIVolumeFinalizerPurgeProtection = Group + "/purge-protection"

	DirectCSIDriveFinalizerDataProtection = Group + "/data-protection"
	DirectCSIDriveFinalizerPrefix         = Group + ".volume/"
)

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   DirectCSIDriveSpec   `json:"spec"`
	Status DirectCSIDriveStatus `json:"status,omitempty"`
}

type DirectCSIDriveSpec struct {
	// +optional
	RequestedFormat *RequestedFormat `json:"requestedFormat,omitempty"`
	// required
	DirectCSIOwned bool `json:"directCSIOwned"`
	// +optional
	DriveTaint map[string]string `json:"driveTaint,omitempty"`
}

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
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

type DirectCSIDriveCondition string

const (
	DirectCSIDriveConditionOwned       DirectCSIDriveCondition = "Owned"
	DirectCSIDriveConditionMounted                             = "Mounted"
	DirectCSIDriveConditionFormatted                           = "Formatted"
	DirectCSIDriveConditionInitialized                         = "Initialized"
)

type DirectCSIDriveReason string

const (
	DirectCSIDriveReasonNotAdded    DirectCSIDriveReason = "NotAdded"
	DirectCSIDriveReasonAdded                            = "Added"
	DirectCSIDriveReasonInitialized                      = "Initialized"
)

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

type DriveStatus string

const (
	DriveStatusInUse       DriveStatus = "InUse"
	DriveStatusAvailable               = "Available"
	DriveStatusUnavailable             = "Unavailable"
	DriveStatusReady                   = "Ready"
	DriveStatusTerminating             = "Terminating"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIDriveList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectCSIDrive `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

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
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status DirectCSIVolumeStatus `json:"status,omitempty"`
}

type DirectCSIVolumeCondition string

const (
	DirectCSIVolumeConditionPublished DirectCSIVolumeCondition = "Published"
	DirectCSIVolumeConditionStaged                             = "Staged"
)

type DirectCSIVolumeReason string

const (
	DirectCSIVolumeReasonNotInUse DirectCSIVolumeReason = "NotInUse"
	DirectCSIVolumeReasonInUse                          = "InUse"
)

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
