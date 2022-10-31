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
	"github.com/minio/directpv/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	volumeFinalizerPVProtection    = Group + "/pv-protection"
	volumeFinalizerPurgeProtection = Group + "/purge-protection"
)

// VolumeStatus denotes volume information.
type VolumeStatus struct {
	DataPath          string             `json:"dataPath"`
	StagingTargetPath string             `json:"stagingTargetPath"`
	TargetPath        string             `json:"targetPath"`
	FSUUID            string             `json:"fsuuid"`
	TotalCapacity     int64              `json:"totalCapacity"`
	AvailableCapacity int64              `json:"availableCapacity"`
	UsedCapacity      int64              `json:"usedCapacity"`
	Status            types.VolumeStatus `json:"status"`
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

	Status VolumeStatus `json:"status"`
}

// NewDirectPVVolume creates new DirectPV volume.
func NewDirectPVVolume(
	name string,
	fsuuid string,
	nodeID types.NodeID,
	driveID types.DriveID,
	driveName types.DriveName,
	size int64,
) *DirectPVVolume {
	return &DirectPVVolume{
		TypeMeta: metav1.TypeMeta{
			APIVersion: Group + "/" + Version,
			Kind:       consts.VolumeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Finalizers: []string{
				volumeFinalizerPVProtection,
				volumeFinalizerPurgeProtection,
			},
			Labels: map[string]string{
				string(types.DriveLabelKey):     string(driveID),
				string(types.NodeLabelKey):      string(nodeID),
				string(types.DriveNameLabelKey): string(driveName),
				string(types.VersionLabelKey):   Version,
				string(types.CreatedByLabelKey): consts.ControllerName,
			},
		},
		Status: VolumeStatus{
			FSUUID:            fsuuid,
			TotalCapacity:     size,
			AvailableCapacity: size,
			Status:            types.VolumeStatusPending,
		},
	}
}

// IsStaged returns whether this volume is staged or not.
func (volume DirectPVVolume) IsStaged() bool {
	return volume.Status.StagingTargetPath != ""
}

// IsPublished returns whether this volume is published or not.
func (volume DirectPVVolume) IsPublished() bool {
	return volume.Status.TargetPath != ""
}

// IsDriveLost returns whether associated drive is lost or not.
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

// SetDriveLost sets associated drive is lost.
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

// IsReleased returns whether this volume is released or not.
func (volume DirectPVVolume) IsReleased() bool {
	return len(volume.Finalizers) == 1 && volume.Finalizers[0] == volumeFinalizerPurgeProtection
}

func (volume *DirectPVVolume) removeFinalizer(value string) {
	finalizers := []string{}
	for _, finalizer := range volume.Finalizers {
		if finalizer != value {
			finalizers = append(finalizers, finalizer)
		}
	}

	if len(finalizers) != len(volume.Finalizers) {
		volume.Finalizers = finalizers
	}
}

// RemovePurgeProtection removes purge protection.
func (volume *DirectPVVolume) RemovePurgeProtection() {
	volume.removeFinalizer(volumeFinalizerPurgeProtection)
}

// RemovePVProtection removes PV protection.
func (volume *DirectPVVolume) RemovePVProtection() {
	volume.removeFinalizer(volumeFinalizerPVProtection)
}

// CopyLabels copies labels from another volumes.
func (volume *DirectPVVolume) CopyLabels(vol *DirectPVVolume) {
	for key, value := range vol.Labels {
		volume.Labels[key] = value
	}
}

// SetLabel sets label to this volume.
func (volume *DirectPVVolume) SetLabel(key types.LabelKey, value types.LabelValue) {
	values := volume.GetLabels()
	if values == nil {
		values = map[string]string{}
	}
	values[string(key)] = string(value)
	volume.SetLabels(values)
}

func (volume DirectPVVolume) getLabel(key types.LabelKey) types.LabelValue {
	values := volume.GetLabels()
	if values == nil {
		values = map[string]string{}
	}
	return types.NewLabelValue(values[string(key)])
}

// SetDriveID sets drive ID of associated drive to this volume.
func (volume *DirectPVVolume) SetDriveID(name types.DriveID) {
	volume.SetLabel(types.DriveLabelKey, types.NewLabelValue(string(name)))
}

// GetDriveID returns drive ID associated drive of this volume.
func (volume DirectPVVolume) GetDriveID() types.DriveID {
	return types.DriveID(volume.getLabel(types.DriveLabelKey))
}

// SetDriveName sets drive name of associated drive to this volume.
func (volume *DirectPVVolume) SetDriveName(name types.DriveName) {
	volume.SetLabel(types.DriveNameLabelKey, types.NewLabelValue(string(name)))
}

// GetDriveName returns drive name of associated drive of this volume.
func (volume DirectPVVolume) GetDriveName() types.DriveName {
	return types.DriveName(volume.getLabel(types.DriveNameLabelKey))
}

// SetNodeID sets node ID of associated drive to this volume.
func (volume *DirectPVVolume) SetNodeID(name types.NodeID) {
	volume.SetLabel(types.NodeLabelKey, types.NewLabelValue(string(name)))
}

// GetNodeID returns node ID of associated drive of this volume.
func (volume DirectPVVolume) GetNodeID() types.NodeID {
	return types.NodeID(volume.getLabel(types.NodeLabelKey))
}

// SetVersionLabel sets version label to this volume.
func (volume *DirectPVVolume) SetVersionLabel() {
	volume.SetLabel(types.VersionLabelKey, Version)
}

// SetCreatedByLabel sets created-by label to this volume.
func (volume *DirectPVVolume) SetCreatedByLabel() {
	volume.SetLabel(types.CreatedByLabelKey, consts.ControllerName)
}

// SetPodName sets associated pod name to this volume.
func (volume *DirectPVVolume) SetPodName(name string) {
	volume.SetLabel(types.PodNameLabelKey, types.NewLabelValue(name))
}

// GetPodName returns associated pod name of this volume.
func (volume DirectPVVolume) GetPodName() string {
	return string(volume.getLabel(types.PodNameLabelKey))
}

// SetPodNS sets associated pod namespace to this volume.
func (volume *DirectPVVolume) SetPodNS(name string) {
	volume.SetLabel(types.PodNSLabelKey, types.NewLabelValue(name))
}

// GetPodNS returns associated pod namespace of this volume.
func (volume DirectPVVolume) GetPodNS() string {
	return string(volume.getLabel(types.PodNSLabelKey))
}

// GetTenantName returns associated tenant name of this volume.
func (volume DirectPVVolume) GetTenantName() string {
	return string(volume.getLabel(types.LabelKey(Group + "/tenant")))
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
