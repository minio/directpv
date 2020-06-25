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

package volume

import (
	"fmt"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/minio/jbod-csi-driver/pkg/topology"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Volume{}, &VolumeList{})
}

type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	VolumeID           string                       `json:"volumeID"`
	Name               string                       `json:"name,omitempty"`
	VolumeSource       VolumeSource                 `json:"volumeSource,omitempty"`
	VolumeStatus       VolumeStatus                 `json:"volumeStatus"`
	NodeID             string                       `json:"nodeID,omitempty"`
	StagingPath        string                       `json:"stagingPath"`
	VolumeAccessMode   VolumeAccessMode             `json:"volumeAccessMode"`
	BlockAccess        []BlockAccessType            `json:"blockAccess,omitempty"`
	MountAccess        []MountAccessType            `json:"mountAccess,omitempty"`
	PublishContext     map[string]string            `json:"publishContext,omitempty"`
	Parameters         map[string]string            `json:"parameters,omitempty"`
	TopologyConstraint *topology.TopologyConstraint `json:"topologyConstraint,omitempty"`
	AuditTrail         map[time.Time]VolumeStatus   `json:"auditTrail,omitempty"`
}

type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

type VolumeSource struct {
	VolumeSourceType VolumeSourceType `json:"volumeSourceType"`
	VolumeSourcePath string           `json:"volumeSourcePath"`
}

type VolumeSourceType string

const (
	VolumeSourceTypeBlockDevice VolumeSourceType = "BlockDevice"
	VolumeSourceTypeDirectory   VolumeSourceType = "Directory"
)

type VolumeStatus string

const (
	VolumeStatusCreated   VolumeStatus = "created"
	VolumeStatusNodeReady VolumeStatus = "node-ready"
	VolumeStatusVolReady  VolumeStatus = "vol-ready"
	VolumeStatusPublished VolumeStatus = "published"
)

type VolumeAccessMode int

const (
	VolumeAccessModeUnknown VolumeAccessMode = iota
	VolumeAccessModeSingleNodeWriter
	VolumeAccessModeSingleNodeReadOnly
	VolumeAccessModeMultiNodeReadOnly
	VolumeAccessModeMultiNodeSingleWriter
	VolumeAccessModeMultiNodeMultiWriter
)

func (v VolumeAccessMode) IgnoreMarshalJSON() ([]byte, error) {
	switch v {
	case VolumeAccessModeUnknown:
		return []byte("UNKNOWN"), nil
	case VolumeAccessModeSingleNodeWriter:
		return []byte("SINGLE_NODE_WRITER"), nil
	case VolumeAccessModeSingleNodeReadOnly:
		return []byte("SINGLE_NODE_READ_ONLY"), nil
	case VolumeAccessModeMultiNodeReadOnly:
		return []byte("MULTI_NODE_READ_ONLY"), nil
	case VolumeAccessModeMultiNodeSingleWriter:
		return []byte("MULTI_NODE_SINGLE_WRITER"), nil
	case VolumeAccessModeMultiNodeMultiWriter:
		return []byte("MULTI_NODE_MULTI_WRITER"), nil
	default:
		return nil, fmt.Errorf("invalid volume access mode")
	}
}

func (v VolumeAccessMode) IgnoreUnmarshalJSON(value []byte) error {
	switch string(value) {
	case "UNKNOWN":
		v = VolumeAccessModeUnknown
	case "SINGLE_NODE_WRITER":
		v = VolumeAccessModeSingleNodeWriter
	case "SINGLE_NODE_READ_ONLY":
		v = VolumeAccessModeSingleNodeReadOnly
	case "MULTI_NODE_READ_ONLY":
		v = VolumeAccessModeMultiNodeReadOnly
	case "MULTI_NODE_SINGLE_WRITER":
		v = VolumeAccessModeMultiNodeSingleWriter
	case "MULTI_NODE_MULTI_WRITER":
		v = VolumeAccessModeMultiNodeMultiWriter
	default:
		return fmt.Errorf("invalid volume access mode")
	}
	return nil
}

type Access string

const (
	AccessRO Access = "ro"
	AccessRW Access = "rw"
)

type AccessType interface {
	Matches(*csi.NodePublishVolumeRequest) bool
}

type BlockAccessType struct {
	Device string `json:"device"`
	Link   string `json:"link,omitempty"`
	Access Access `json:"access,omitempty"`
}

func (b BlockAccessType) Matches(req *csi.NodePublishVolumeRequest) bool {
	targetPath := req.GetTargetPath()
	ro := req.GetReadonly()

	if targetPath == "" || targetPath != b.Link {
		return false
	}

	switch ro {
	case true:
		if b.Access != AccessRO {
			return false
		}
	case false:
		if b.Access != AccessRW {
			return false
		}
	}

	return true
}

type MountAccessType struct {
	FsType     FsType      `json:"fsType"`
	MountFlags []MountFlag `json:"mountFlags"`
	MountPoint string      `json:"mountpoint"`
	Access     Access      `json:"access,omitempty"`
}

func (m MountAccessType) Matches(req *csi.NodePublishVolumeRequest) bool {
	targetPath := req.GetTargetPath()
	ro := req.GetReadonly()

	if targetPath == "" || targetPath != m.MountPoint {
		return false
	}

	switch ro {
	case true:
		if m.Access != AccessRO {
			return false
		}
	case false:
		if m.Access != AccessRW {
			return false
		}
	}

	if len(m.MountFlags) == 0 && len(m.FsType) == 0 {
		return targetPath == m.MountPoint
	}

	vCap := req.GetVolumeCapability()
	if vCap == nil {
		return false
	}

	vMount := vCap.GetMount()
	if vMount == nil {
		return false
	}

	mFlags := vMount.GetMountFlags()
	fsType := vMount.GetFsType()

	if fsType != string(m.FsType) {
		return false
	}

	listCompare := func(l []string, r []MountFlag) bool {
		lMap := map[string]struct{}{}
		rMap := map[string]struct{}{}

		for _, a := range l {
			lMap[a] = struct{}{}
		}

		for _, b := range r {
			rMap[string(b)] = struct{}{}
		}

		for a := range lMap {
			if _, ok := rMap[a]; !ok {
				return false
			}
		}

		for b := range rMap {
			if _, ok := lMap[b]; !ok {
				return false
			}
		}
		return true
	}

	if !listCompare(mFlags, m.MountFlags) {
		return false
	}

	return true
}

type FsType string

const (
	FsTypeXFS  FsType = "xfs"
	FsTypeExt4 FsType = "ext4"
)

type MountFlag string

const (
	MountFlagRO      MountFlag = "ro"
	MountFlagRemount MountFlag = "remount"
)

type MountOption string

const (
	MountOptionBind MountOption = "bind"
)
