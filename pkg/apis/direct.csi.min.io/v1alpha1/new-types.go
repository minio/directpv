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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=directcsidrive,singular=directcsidrive
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.currentState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type DirectCSIDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	ModelNumber   string                 `json:"modelNumber,omitempty"`
	SerialNumber  string                 `json:"serialNumber,omitempty"`
	Name          string                 `json:"name,omitempty"`
	OwnerNode     corev1.ObjectReference `json:"ownerNode"`
	TotalCapacity resource.Quantity      `json:"totalCapacity"`
	FreeCapacity  resource.Quantity      `json:"freeCapacity"`
	Volumes       []DirectCSIVolume      `json:"volumes"`
	Path          string                 `json:"path"`
	Formatted     bool                   `json:"formatted"`
	RootPartition string                 `json:"rootPartition"`
	PartitionNum  int                    `json:"PartitionNum,omitempty"`
	FileSystem    string                 `json:"filesystem,omitempty"`
	Status        DriveStatus            `json:"driveStatus"`
}

type DriveStatus string

const (
	Online      DriveStatus = "online"
	Offline                 = "offline"
	Unformatted             = "new"
	Other                   = "other"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=directcsivolume,singular=directcsivolume
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.currentState"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type DirectCSIVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	ID            string                 `json:"ID"`
	Name          string                 `json:"name,omitempty"`
	OwnerDrive    DirectCSIDrive         `json:"ownerDrive"`
	OwnerNode     corev1.ObjectReference `json:"ownerNode"`
	SourcePath    string                 `json:"sourcePath"`
	TotalCapacity resource.Quantity      `json:"totalCapacity"`
	VolumeStatus  VolumeStatus           `json:"volumeStatus"`
}
