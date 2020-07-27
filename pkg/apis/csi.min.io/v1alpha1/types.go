// This file is part of MinIO vCenter Plugin
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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient:nonNamespaced
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
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageTopology defines the layout of storage infrastructure.
type StorageTopology struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the storage topology
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status.
	// +optional
	Spec StorageTopologySpec `json:"spec,omitempty"`
	// StorageTopologyStatus defines the observed state of StorageTopology
	// +optional
	Status StorageTopologyStatus `json:"status,omitempty"`
}

// StorageTopologyLayout describes the nodes and drives that will be used across the cluster.
type StorageTopologyLayout struct {
	// Labels of the nodes that will participate
	NodeLabels map[string]string `json:"nodeLabels,omitempty"`
	// Mount Path for the volumes that will be used
	Path string `json:"string"`
}

// StorageTopologySpec is the spec of the StorageTopology.
type StorageTopologySpec struct {
	// Layout of the drives across hosts
	Layout StorageTopologyLayout `json:"layout"`
	// Layout of the drives across hosts
	// +optional
	FsType string `json:"fstype,omitempty"`
	// Mount Options
	// +optional
	MountOptions []string `json:"mount_options,omitempty"`
	// Resource Limits
	// +optional
	ResourceLimits StorageTopologyResourceLimit `json:"resource_limits,omitempty"`
	// Reclaim Policy
	// +optional
	ReclaimPolicy string `json:"reclaim_policy,omitempty"`
}

type StorageTopologyResourceLimit struct {
	// Storage
	// +optional
	Storage resource.Quantity `json:"storage,omitempty"`
	// Volumes
	// +optional
	Volumes int32 `json:"volumes,omitempty"`
}

// StorageTopologyStatus is the state of the StorageTopology
type StorageTopologyStatus struct {
	// Nodes the storage topology covers
	// +optional
	Nodes []string `json:"accessibleNodes,omitempty"`
	// Total Capacity of the storage topology across all nodes
	// +optional
	Capacity StorageTopologyStatusCapacity `json:"capacity,omitempty"`
	// Present only when there is an error.
	// +optional
	Error StorageTopologyStatusError `json:"error,omitempty"`
}

type StorageTopologyStatusCapacity struct {
	// Free Space of the storage topology
	// +optional
	FreeSpace resource.Quantity `json:"freeSpace,omitempty"`
	// Total capacity of the storage topology
	// +optional
	Total resource.Quantity `json:"total,omitempty"`
}

type StorageTopologyStatusError struct {
	// Message details of the encountered error
	// +optional
	Message string `json:"message,omitempty"`
	// State indicates a single word description of the error state that has occurred on the StorageTopology, "InMaintenance",
	// "NotAccessible", etc.
	// +optional
	State string `json:"state,omitempty"`
}
