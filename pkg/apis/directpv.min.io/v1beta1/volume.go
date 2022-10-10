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

// DirectPVVolumeStatus denotes volume information.
type DirectPVVolumeStatus struct {
	DataPath          string `json:"dataPath"`
	StagingTargetPath string `json:"stagingTargetPath"`
	TargetPath        string `json:"targetPath"`
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

func (volume DirectPVVolume) IsStaged() bool {
	return volume.Status.StagingTargetPath != ""
}

func (volume DirectPVVolume) IsPublished() bool {
	return volume.Status.TargetPath != ""
}

func (volume DirectPVVolume) IsDriveLost() bool {
	for _, condition := range volume.Status.Conditions {
		if condition.Type == string(types.VolumeConditionTypeLost) &&
			condition.Status == metav1.ConditionTrue &&
			condition.Reason == string(types.VolumeConditionReasonDriveLost) &&
			condition.Message == string(types.VolumeConditionMessageDriveLost) {
			return true
		}
	}

	return false
}

func (volume *DirectPVVolume) SetDriveLost() {
	c := metav1.Condition{
		Type:               string(types.VolumeConditionTypeLost),
		Status:             metav1.ConditionTrue,
		Reason:             string(types.VolumeConditionReasonDriveLost),
		Message:            string(types.VolumeConditionMessageDriveLost),
		LastTransitionTime: metav1.Now(),
	}
	updated := false
	for i := range volume.Status.Conditions {
		if volume.Status.Conditions[i].Type == string(types.VolumeConditionTypeLost) {
			volume.Status.Conditions[i] = c
			updated = true
			break
		}
	}
	if !updated {
		volume.Status.Conditions = append(volume.Status.Conditions, c)
	}
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
