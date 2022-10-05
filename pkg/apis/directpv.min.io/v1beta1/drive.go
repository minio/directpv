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

import (
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DirectPVDriveStatus denotes drive information.
type DirectPVDriveStatus struct {
	Path              string            `json:"path"`
	TotalCapacity     int64             `json:"totalCapacity"`
	AllocatedCapacity int64             `json:"allocatedCapacity"`
	FreeCapacity      int64             `json:"freeCapacity"`
	FSUUID            string            `json:"fsuuid"`
	NodeName          string            `json:"nodeName"`
	Status            types.DriveStatus `json:"status"`
	Topology          map[string]string `json:"topology"`
	// +optional
	ModelNumber string `json:"modelNumber,omitempty"`
	// +optional
	Vendor string `json:"vendor,omitempty"`
	// +optional
	AccessTier types.AccessTier `json:"accessTier,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVDrive denotes drive CRD object.
type DirectPVDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Status DirectPVDriveStatus `json:"status"`
}

func (drive DirectPVDrive) IsLost() bool {
	return drive.Status.Status == types.DriveStatusLost
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVDriveList denotes list of drives.
type DirectPVDriveList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectPVDrive `json:"items"`
}
