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
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

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
	Paths []string `json:"paths"`
}

// StorageTopologySpec is the spec of the StorageTopology.
type StorageTopologySpec struct {
	// Layout of the drives across hosts
	Layout StorageTopologyLayout `json:"layout"`
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

type ReclaimPolicy string

const (
	ReclaimPolicyRetain ReclaimPolicy = "Retain"
	ReclaimPolicyDelete ReclaimPolicy = "Delete"
)

// StorageLimit describes maximum limits on the storage provisioned in this topology
type StorageLimit struct {
	// Storage represents the maximum size of a volume that can be provisioned in this topology
	// +optional
	MaxVolumeSize resource.Quantity `json:"maxVolumeSize,omitempty"`
	// MaxVolumeCount describes the maximum number of volumes allowed to be provisioned in this topology
	// +optional
	MaxVolumeCount int32 `json:"maxVolumeCount,omitempty"`
}

// StorageTopologyStatus is the state of the StorageTopology
type StorageTopologyStatus struct {
	// Nodes the storage topology covers
	// +optional
	Nodes []StorageTopologyNodeStatus `json:"nodes,omitempty"`
	// Paths is the list of paths after ellipses expansion
	Paths []string `json:"paths"`
	// TotalAvailableSize is the used capacity across all nodes
	// +optional
	TotalAvailableSize resource.Quantity `json:"totalAvailableSize,omitempty"`
	// TotalAvailableCount is the number of allocated drives
	// +optional
	TotalAvailableCount int32 `json:"totalAvailableCount,omitempty"`
	// TotalAllocatedSize is the used capacity across all nodes
	// +optional
	TotalAllocatedSize resource.Quantity `json:"totalAllocatedSize,omitempty"`
	// TotalAllocatedCount is the number of allocated drives
	// +optional
	TotalAllocatedCount int32 `json:"totalAllocatedCount,omitempty"`
	// Describes the readiness of this topology
	// +optional
	Conditions []StorageTopologyConditions `json:"conditions,omitempty"`
}

// StorageTopologyNodeStatus describes the status of the nodes that fall under this particular topology
type StorageTopologyNodeStatus struct {
	// NodeName is the name of the node
	NodeName string `json:"nodeName"`
	// allocatedSize is the used capacity on this node
	// +optional
	AllocatedSize resource.Quantity `json:"volumesAllocatedSize,omitempty"`
	// Number of allocated drives
	// +optional
	AllocatedCount int32 `json:"volumesAllocatedCount,omitempty"`
	// Capacity of the storage topology available on this node
	// +optional
	AvailableSize resource.Quantity `json:"volumesAvailableSize,omitempty"`
	// Number of available drives
	// +optional
	AvailableCount int32 `json:"volumesAvailableCount,omitempty"`
	// Condition describes the readiness of this node for this storage topology
	// +optional
	Conditions []StorageTopologyConditions `json:"conditions,omitempty"`
}

// StorageTopologyConditions describes the trueness of a condition for the StorageTopology
type StorageTopologyConditions struct {
	// Condition is a one word description of the represented value
	Condition StorageTopologyCondition `json:"condition"`

	// Status indicates if the condition is true or false
	Status bool `json:"status"`

	// Message about the conditions value
	// +optional
	Message string `json:"message,omitempty"`
}

type StorageTopologyCondition string

const (
	StorageTopologyConditionNodeReady StorageTopologyCondition = "NodeReady"
	StorageTopologyConditionAllNodesReady StorageTopologyCondition = "AllNodesReady"
)
