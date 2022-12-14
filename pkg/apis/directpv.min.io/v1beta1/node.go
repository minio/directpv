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
	"github.com/minio/directpv/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVNodeList denotes list of nodes.
type DirectPVNodeList struct {
	metav1.TypeMeta `json:",inline"`
	// metdata is the standard list metadata.
	// +optional
	metav1.ListMeta `json:"metadata"`
	Items           []DirectPVNode `json:"items"`
}

// +genclient
// +genclient:nonNamespaced
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DirectPVNode denotes Node CRD object.
type DirectPVNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status"`
}

// GetDevicesByNames fetchs the devices in the node by device names.
func (node DirectPVNode) GetDevicesByNames(names []string) (devices []Device) {
	if len(names) == 0 {
		return node.Status.Devices
	}
	for i := range node.Status.Devices {
		if utils.Contains(names, node.Status.Devices[i].Name) {
			devices = append(devices, node.Status.Devices[i])
		}
	}
	return
}

// NewDirectPVNode creates new DirectPV node.
func NewDirectPVNode(
	nodeID types.NodeID,
	devices []Device,
) *DirectPVNode {
	return &DirectPVNode{
		TypeMeta: metav1.TypeMeta{
			APIVersion: Group + "/" + Version,
			Kind:       consts.NodeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: string(nodeID),
			Labels: map[string]string{
				string(types.NodeLabelKey):      string(nodeID),
				string(types.VersionLabelKey):   Version,
				string(types.CreatedByLabelKey): consts.NodeControllerName,
			},
		},
		Spec: NodeSpec{
			Refresh: false,
		},
		Status: NodeStatus{
			Devices: devices,
		},
	}
}

// NodeSpec represents DirectPV node specification values.
type NodeSpec struct {
	// +optional
	Refresh bool `json:"refresh,omitempty"`
}

// NodeStatus denotes node information.
type NodeStatus struct {
	// +listType=atomic
	Devices []Device `json:"devices"`
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// Device denotes the device information in a drive
type Device struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	MajorMinor string `json:"majorMinor"`
	Size       uint64 `json:"size"`
	// +optional
	Make string `json:"make,omitempty"`
	// +optional
	FSType string `json:"fsType,omitempty"`
	// +optional
	FSUUID string `json:"fsuuid,omitempty"`
	// +optional
	DeniedReason string `json:"deniedReason,omitempty"`
}
