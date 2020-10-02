// This file is part of MinIO Kubernetes Cloud
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

type ReclaimPolicy string

const (
	ReclaimPolicyRetain ReclaimPolicy = "Retain"
	ReclaimPolicyDelete ReclaimPolicy = "Delete"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageTopologyList is a list of StorageTopology objects.
type StorageTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`

	// items is the list of Storage Topology objects.
	Items []StorageTopology `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageTopology defines the layout of storage infrastructure.
type StorageTopology struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Layout of the drives across hosts
	Layout []StorageTopologyLayout `json:"layout"`
	// Layout of the drives across hosts
	// +optional
	FsType string `json:"fstype,omitempty"`
	// Mount Options describes the options to be passed in for mounting the direct attached storage into pods
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`
	// Limits describes the limits on the maximum values of the volumes in this topology
	// +optional
	Limits StorageLimit `json:"limits,omitempty"`
	// Reclaim Policy defines the default reclaim policy of volumes in this topology
	// +optional
	ReclaimPolicy ReclaimPolicy `json:"reclaimPolicy,omitempty"`
}

// StorageTopologyLayout describes the nodes and drives that will be used across the cluster.
type StorageTopologyLayout struct {
	// Selector for the nodes that will participate
	NodeSelector map[string]string `json:"nodeLabels,omitempty"`
	// Drive Paths from which volumes will be carved out
	Blkid []string `json:"blkid"`
}

// StorageLimit describes maximum limits on the storage provisioned in this topology
type StorageLimit struct {
	// MaxVolumeCount describes the maximum number of volumes allowed to be provisioned in this topology
	// +optional
	MaxVolumeCount int32 `json:"maxVolumeCount,omitempty"`
}
