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

type DriveSpec struct {
	// +optional
	Unschedulable bool `json:"unschedulable,omitempty"`
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

func (drive *DirectPVDrive) Unschedulable() {
	drive.Spec.Unschedulable = true
}

func (drive *DirectPVDrive) Schedulable() {
	drive.Spec.Unschedulable = false
}

func (drive DirectPVDrive) IsUnschedulable() bool {
	return drive.Spec.Unschedulable
}

func (drive DirectPVDrive) GetDriveID() types.DriveID {
	return types.DriveID(drive.Name)
}

func (drive DirectPVDrive) GetVolumeCount() int {
	return len(drive.Finalizers) - 1
}

func (drive DirectPVDrive) VolumeExist(volume string) bool {
	return utils.Contains(drive.Finalizers, driveFinalizerVolumePrefix+volume)
}

func (drive DirectPVDrive) GetVolumes() (names []string) {
	for _, finalizer := range drive.Finalizers {
		if strings.HasPrefix(finalizer, driveFinalizerVolumePrefix) {
			names = append(names, strings.TrimPrefix(finalizer, driveFinalizerVolumePrefix))
		}
	}
	return
}

func (drive *DirectPVDrive) ResetFinalizers() {
	drive.Finalizers = []string{driveFinalizerDataProtection}
}

func (drive *DirectPVDrive) RemoveFinalizers() bool {
	if len(drive.Finalizers) == 1 && drive.Finalizers[0] == driveFinalizerDataProtection {
		drive.Finalizers = []string{}
		return true
	}
	return false
}

func (drive *DirectPVDrive) AddVolumeFinalizer(volume string) (added bool) {
	value := driveFinalizerVolumePrefix + volume
	for _, finalizer := range drive.Finalizers {
		if finalizer == string(value) {
			return false
		}
	}

	drive.Finalizers = append(drive.Finalizers, string(value))
	return true
}

func (drive *DirectPVDrive) RemoveVolumeFinalizer(volume string) (found bool) {
	value := driveFinalizerVolumePrefix + volume
	finalizers := []string{}
	for _, finalizer := range drive.Finalizers {
		if finalizer == string(value) {
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

func (drive *DirectPVDrive) setLabel(key types.LabelKey, value types.LabelValue) {
	values := drive.GetLabels()
	if values == nil {
		values = map[string]string{}
	}
	values[string(key)] = string(value)
	drive.SetLabels(values)
}

func (drive DirectPVDrive) getLabel(key types.LabelKey) types.LabelValue {
	values := drive.GetLabels()
	if values == nil {
		values = map[string]string{}
	}
	return types.NewLabelValue(values[string(key)])
}

func (drive *DirectPVDrive) SetDriveName(name types.DriveName) {
	drive.setLabel(types.DriveNameLabelKey, types.NewLabelValue(string(name)))
}

func (drive DirectPVDrive) GetDriveName() types.DriveName {
	return types.DriveName(drive.getLabel(types.DriveNameLabelKey))
}

func (drive *DirectPVDrive) SetNodeID(name types.NodeID) {
	drive.setLabel(types.NodeLabelKey, types.NewLabelValue(string(name)))
}

func (drive DirectPVDrive) GetNodeID() types.NodeID {
	return types.NodeID(drive.getLabel(types.NodeLabelKey))
}

func (drive *DirectPVDrive) SetAccessTier(value types.AccessTier) {
	drive.setLabel(types.AccessTierLabelKey, types.NewLabelValue(string(value)))
}

func (drive DirectPVDrive) GetAccessTier() types.AccessTier {
	return types.AccessTier(drive.getLabel(types.AccessTierLabelKey))
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
