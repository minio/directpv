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
	"github.com/google/uuid"
	"github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVInitRequestList denotes list of init request.
type DirectPVInitRequestList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectPVInitRequest `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVInitRequest denotes DirectPVInitRequest CRD object.
type DirectPVInitRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   InitRequestSpec   `json:"spec"`
	Status InitRequestStatus `json:"status"`
}

func (req DirectPVInitRequest) getLabel(key types.LabelKey) types.LabelValue {
	values := req.GetLabels()
	return types.ToLabelValue(values[string(key)])
}

// NewDirectPVInitRequest creates new DirectPV init request.
func NewDirectPVInitRequest(
	requestID string,
	nodeID types.NodeID,
	devices []InitDevice,
) *DirectPVInitRequest {
	return &DirectPVInitRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: Group + "/" + Version,
			Kind:       consts.InitRequestKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
			Labels: map[string]string{
				string(types.NodeLabelKey):      string(nodeID),
				string(types.VersionLabelKey):   Version,
				string(types.CreatedByLabelKey): consts.NodeControllerName,
				string(types.RequestIDLabelKey): requestID,
			},
		},
		Spec: InitRequestSpec{
			Devices: devices,
		},
		Status: InitRequestStatus{
			Status:  types.InitStatusPending,
			Results: []InitDeviceResult{},
		},
	}
}

// GetNodeID returns node ID of this initrequest.
func (req DirectPVInitRequest) GetNodeID() types.NodeID {
	return types.NodeID(req.getLabel(types.NodeLabelKey))
}

// InitRequestSpec represents the spec for InitRequest.
type InitRequestSpec struct {
	// +listType=atomic
	Devices []InitDevice `json:"devices"`
}

// InitDevice represents the device requested for initialization.
type InitDevice struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Force bool   `json:"force"`
}

// InitRequestStatus represents the status of the InitRequest.
type InitRequestStatus struct {
	Status types.InitStatus `json:"status"`
	// +listType=atomic
	Results []InitDeviceResult `json:"results"`
}

// InitDeviceResult represents the result of the InitDeviceRequest.
type InitDeviceResult struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}
