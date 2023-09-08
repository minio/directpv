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
	"strconv"
	"strings"

	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	driveFinalizerDataProtection = Group + "/data-protection"
	driveFinalizerVolumePrefix   = Group + ".volume/"
)

// DriveSpec represents DirectPV drive specification values.
type DriveSpec struct {
	// +optional
	Unschedulable bool `json:"unschedulable,omitempty"`
	// +optional
	Relabel bool `json:"relabel,omitempty"`
}

// DriveStatus denotes drive information.
type DriveStatus struct {
	TotalCapacity     int64             `json:"totalCapacity"`
	AllocatedCapacity int64             `json:"allocatedCapacity"`
	FreeCapacity      int64             `json:"freeCapacity"`
	FSUUID            string            `json:"fsuuid"`
	Status            types.DriveStatus `json:"status"`
	Topology          map[string]string `json:"topology"`
	// +optional
	Make string `json:"make,omitempty"`
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

// DirectPVDrive denotes drive CRD object.
type DirectPVDrive struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   DriveSpec   `json:"spec,omitempty"`
	Status DriveStatus `json:"status"`
}

// NewDirectPVDrive creates new DirectPV drive.
func NewDirectPVDrive(
	driveID types.DriveID,
	status DriveStatus,
	nodeID types.NodeID,
	driveName types.DriveName,
	accessTier types.AccessTier,
) *DirectPVDrive {
	return &DirectPVDrive{
		TypeMeta: metav1.TypeMeta{
			APIVersion: Group + "/" + Version,
			Kind:       consts.DriveKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       string(driveID),
			Finalizers: []string{driveFinalizerDataProtection},
			Labels: map[string]string{
				string(types.NodeLabelKey):       string(nodeID),
				string(types.DriveNameLabelKey):  string(driveName),
				string(types.AccessTierLabelKey): string(accessTier),
				string(types.VersionLabelKey):    Version,
				string(types.CreatedByLabelKey):  consts.DriverName,
			},
		},
		Status: status,
	}
}

// Unschedulable marks this drive not to schedule volumes.
func (drive *DirectPVDrive) Unschedulable() {
	drive.Spec.Unschedulable = true
}

// Schedulable marks this drive to schedule volumes.
func (drive *DirectPVDrive) Schedulable() {
	drive.Spec.Unschedulable = false
}

// IsUnschedulable returns whether this drive is in unschedulable state.
func (drive DirectPVDrive) IsUnschedulable() bool {
	return drive.Spec.Unschedulable
}

// GetDriveID returns this drive's ID.
func (drive DirectPVDrive) GetDriveID() types.DriveID {
	return types.DriveID(drive.Name)
}

// GetVolumeCount returns number of volumes on this drive.
func (drive DirectPVDrive) GetVolumeCount() int {
	return len(drive.Finalizers) - 1
}

// VolumeExist returns whether given volume is on this drive or not.
func (drive DirectPVDrive) VolumeExist(volume string) bool {
	return utils.Contains(drive.Finalizers, driveFinalizerVolumePrefix+volume)
}

// GetVolumes returns volume names on this drive.
func (drive DirectPVDrive) GetVolumes() (names []string) {
	for _, finalizer := range drive.Finalizers {
		if strings.HasPrefix(finalizer, driveFinalizerVolumePrefix) {
			names = append(names, strings.TrimPrefix(finalizer, driveFinalizerVolumePrefix))
		}
	}
	return
}

// ResetFinalizers removes all volume finalizers.
func (drive *DirectPVDrive) ResetFinalizers() {
	drive.Finalizers = []string{driveFinalizerDataProtection}
}

// RemoveFinalizers removes finalizers.
func (drive *DirectPVDrive) RemoveFinalizers() bool {
	if len(drive.Finalizers) == 1 && drive.Finalizers[0] == driveFinalizerDataProtection {
		drive.Finalizers = []string{}
		return true
	}
	return false
}

// AddVolumeFinalizer adds volume to this drive's finalizer.
func (drive *DirectPVDrive) AddVolumeFinalizer(volume string) (added bool) {
	value := driveFinalizerVolumePrefix + volume
	for _, finalizer := range drive.Finalizers {
		if finalizer == value {
			return false
		}
	}

	drive.Finalizers = append(drive.Finalizers, value)
	return true
}

// RemoveVolumeFinalizer remove volume from this drive's finalizer.
func (drive *DirectPVDrive) RemoveVolumeFinalizer(volume string) (found bool) {
	value := driveFinalizerVolumePrefix + volume
	finalizers := []string{}
	for _, finalizer := range drive.Finalizers {
		if finalizer == value {
			found = true
		} else {
			finalizers = append(finalizers, finalizer)
		}
	}

	if found {
		drive.Finalizers = finalizers
	}

	return
}

// GetLabels overrides the definition to return non-nil map.
func (drive *DirectPVDrive) GetLabels() map[string]string {
	values := drive.ObjectMeta.GetLabels()
	if values == nil {
		values = map[string]string{}
		drive.SetLabels(values)
	}
	return values
}

func (drive DirectPVDrive) getLabel(key types.LabelKey) types.LabelValue {
	values := drive.GetLabels()
	return types.ToLabelValue(values[string(key)])
}

// SetDriveName sets name to this drive.
func (drive *DirectPVDrive) SetDriveName(name types.DriveName) {
	drive.SetLabel(types.DriveNameLabelKey, types.ToLabelValue(string(name)))
}

// GetDriveName returns name of this drive.
func (drive DirectPVDrive) GetDriveName() types.DriveName {
	return types.DriveName(drive.getLabel(types.DriveNameLabelKey))
}

// SetNodeID sets node ID to this drive.
func (drive *DirectPVDrive) SetNodeID(name types.NodeID) {
	drive.SetLabel(types.NodeLabelKey, types.ToLabelValue(string(name)))
}

// GetNodeID returns node ID of this drive.
func (drive DirectPVDrive) GetNodeID() types.NodeID {
	return types.NodeID(drive.getLabel(types.NodeLabelKey))
}

// HasVolumeClaimID checks if the provided volume claim id is set on the drive.
func (drive *DirectPVDrive) HasVolumeClaimID(claimID string) bool {
	if claimID == "" {
		return false
	}
	return drive.GetLabels()[types.VolumeClaimIDLabelKeyPrefix+claimID] == strconv.FormatBool(true)
}

// SetVolumeClaimID sets the provided claim id on the drive.
func (drive *DirectPVDrive) SetVolumeClaimID(claimID string) {
	if claimID == "" {
		return
	}
	drive.SetLabel(types.LabelKey(types.VolumeClaimIDLabelKeyPrefix+claimID), types.LabelValue(strconv.FormatBool(true)))
}

// RemoveVolumeClaimID removes the volume claim id label.
func (drive *DirectPVDrive) RemoveVolumeClaimID(claimID string) {
	if claimID == "" {
		return
	}
	drive.RemoveLabel(types.LabelKey(types.VolumeClaimIDLabelKeyPrefix + claimID))
}

// SetLabel sets label to this drive.
func (drive *DirectPVDrive) SetLabel(key types.LabelKey, value types.LabelValue) bool {
	values := drive.GetLabels()
	if v, ok := values[string(key)]; ok && v == string(value) {
		return false
	}
	values[string(key)] = string(value)
	return true
}

// RemoveLabel unsets the label from this drive.
func (drive *DirectPVDrive) RemoveLabel(key types.LabelKey) (found bool) {
	labels := drive.GetLabels()
	_, found = labels[string(key)]
	delete(labels, string(key))
	return
}

// GetAccessTier returns access-tier of this drive.
func (drive DirectPVDrive) GetAccessTier() types.AccessTier {
	return types.AccessTier(drive.getLabel(types.AccessTierLabelKey))
}

// SetMountErrorCondition sets mount error condition to this drive.
func (drive *DirectPVDrive) SetMountErrorCondition(message string) {
	drive.setErrorCondition(string(types.DriveConditionTypeMountError), string(types.DriveConditionReasonMountError), message)
}

// SetMultipleMatchesErrorCondition sets multiple matches error condition to this drive.
func (drive *DirectPVDrive) SetMultipleMatchesErrorCondition(message string) {
	drive.setErrorCondition(string(types.DriveConditionTypeMultipleMatches), string(types.DriveConditionReasonMultipleMatches), message)
}

// SetIOErrorCondition sets I/O error condition to this drive.
func (drive *DirectPVDrive) SetIOErrorCondition() {
	drive.setErrorCondition(string(types.DriveConditionTypeIOError), string(types.DriveConditionReasonIOError), string(types.DriveConditionMessageIOError))
}

// SetRelabelErrorCondition sets relabel error error condition to this drive.
func (drive *DirectPVDrive) SetRelabelErrorCondition(message string) {
	drive.setErrorCondition(string(types.DriveConditionTypeRelabelError), string(types.DriveConditionReasonRelabelError), message)
}

func (drive *DirectPVDrive) setErrorCondition(errType, reason, message string) {
	c := metav1.Condition{
		Type:               errType,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	updated := false
	for i := range drive.Status.Conditions {
		if drive.Status.Conditions[i].Type == errType {
			drive.Status.Conditions[i] = c
			updated = true
			break
		}
	}
	if !updated {
		drive.Status.Conditions = append(drive.Status.Conditions, c)
	}
}

// GetLatestErrorConditionType returns the latest error condition type set.
func (drive *DirectPVDrive) GetLatestErrorConditionType() (errType types.DriveConditionType) {
	var latestCondition *metav1.Condition
	for i := range drive.Status.Conditions {
		switch types.DriveConditionType(drive.Status.Conditions[i].Type) {
		case types.DriveConditionTypeMountError, types.DriveConditionTypeMultipleMatches, types.DriveConditionTypeIOError, types.DriveConditionTypeRelabelError:
			if latestCondition == nil || drive.Status.Conditions[i].LastTransitionTime.After(latestCondition.LastTransitionTime.Time) {
				latestCondition = &drive.Status.Conditions[i]
			}
		}
	}

	if latestCondition != nil {
		errType = types.DriveConditionType(latestCondition.Type)
	}

	return
}

// SetMigratedLabel sets migrated label to this drive.
func (drive *DirectPVDrive) SetMigratedLabel() {
	drive.SetLabel(types.MigratedLabelKey, "true")
}

// IsMigrated indicates whether this is migrated drive or not.
func (drive *DirectPVDrive) IsMigrated() bool {
	return drive.getLabel(types.MigratedLabelKey) == "true"
}

// IsSuspended returns if the drive is suspended.
func (drive DirectPVDrive) IsSuspended() bool {
	return string(drive.getLabel(types.SuspendLabelKey)) == strconv.FormatBool(true)
}

// Suspend suspends the drive by setting the label `directpv.min.io/suspend: true`.
func (drive *DirectPVDrive) Suspend() bool {
	return drive.SetLabel(types.SuspendLabelKey, types.ToLabelValue(strconv.FormatBool(true)))
}

// Resume reverts the suspended drive by removing the label `directpv.min.io/suspend`.
func (drive *DirectPVDrive) Resume() bool {
	return drive.RemoveLabel(types.SuspendLabelKey)
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
