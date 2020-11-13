// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	OwnerNode string `json:"ownerNode"`

	DirectCSIOwned bool `json:"directCSIOwned"`

	Path string `json:"path"`
	// +optional
	ModelNumber string `json:"modelNumber,omitempty"`
	// +optional
	SerialNumber string `json:"serialNumber,omitempty"`
	// +optional
	TotalCapacity int64 `json:"totalCapacity,omitempty"`
	// +optional
	AllocatedCapacity int64 `json:"allocatedCapacity,omitempty"`
	// +optional
	FreeCapacity int64 `json:"freeCapacity,omitempty"`
	// +optional
	BlockSize int64 `json:"blockSize,omitempty"`
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
	DriveStatus DriveStatus `json:"driveStatus,omitempty"`
	// +optional
	Topology map[string]string `json:"topology,omitempty"`
	// +optional
	RequestedFormat RequestedFormat `json:"requestedFormat,omitempty"`
}

type RequestedFormat struct {
	// +optional
	Force bool `json:"force,omitempty"`
	// +optional
	Filesystem string `json:"filesystem,omitempty"`
	// +optional
	Mountpoint string `json:"mountpoint,omitempty"`
}

type DriveStatus string

const (
	Online      DriveStatus = "online"
	Offline                 = "offline"
	Unformatted             = "new"
	Other                   = "other"
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
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +optional
	OwnerDrive string `json:"ownerDrive,omitempty"`
	// +optional
	OwnerNode string `json:"ownerNode,omitempty"`
	// +optional
	HostPath string `json:"hostPath,omitempty"`
	// +optional
	StagingPath string `json:"stagingPath,omitempty"`
	// +optional
	ContainerPath string `json:"containerPath,omitempty"`
	// +optional
	TotalCapacity int64 `json:"totalCapacity"`
}
