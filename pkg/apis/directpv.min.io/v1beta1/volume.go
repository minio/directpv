// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DirectPVVolumeStatus denotes volume information.
type DirectPVVolumeStatus struct {
	HostPath          string `json:"hostPath"`
	StagingPath       string `json:"stagingPath"`
	ContainerPath     string `json:"containerPath"`
	DriveName         string `json:"driveName"`
	FSUUID            string `json:"fsuuid"`
	NodeName          string `json:"nodeName"`
	TotalCapacity     int64  `json:"totalCapacity"`
	AvailableCapacity int64  `json:"availableCapacity"`
	UsedCapacity      int64  `json:"usedCapacity"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVVolume denotes volume CRD object.
type DirectPVVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status DirectPVVolumeStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVVolumeList denotes list of volumes.
type DirectPVVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectPVVolume `json:"items"`
}
