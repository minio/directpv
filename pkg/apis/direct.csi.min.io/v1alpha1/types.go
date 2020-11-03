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

	ModelNumber   string      `json:"modelNumber,omitempty"`
	SerialNumber  string      `json:"serialNumber,omitempty"`
	Name          string      `json:"name,omitempty"`
	OwnerNode     string      `json:"ownerNode,omitempty"`
	TotalCapacity int64       `json:"totalCapacity,omitempty"`
	FreeCapacity  int64       `json:"freeCapacity,omitempty"`
	BlockSize     int64       `json:"blockSize,omitempty"`
	Path          string      `json:"path,omitempty"`
	RootPartition string      `json:"rootPartition,omitempty"`
	PartitionNum  int         `json:"partitionNum,omitempty"`
	Filesystem    string      `json:"filesystem,omitempty"`
	Mountpoint    string      `json:"mountpoint,omitempty"`
	MountOptions  []string    `json:"mountOptions,omitempty"`
	Status        DriveStatus `json:"driveStatus,omitempty"`
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

	// items is the list of DirectCSIDrive objects.
	Items []DirectCSIDrive `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DirectCSIVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`

	// items is the list of Storage Topology objects.
	Items []DirectCSIVolume `json:"items"`
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

	ID            string          `json:"ID"`
	Name          string          `json:"name,omitempty"`
	OwnerDrive    *DirectCSIDrive `json:"ownerDrive,omitempty"`
	OwnerNode     string          `json:"ownerNode,omitempty"`
	SourcePath    string          `json:"sourcePath"`
	TotalCapacity int64           `json:"totalCapacity"`
}
