// This file is part of MinIO
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

// AUTO GENERATED CODE. DO NOT EDIT.

package types

import (
	directpv "github.com/minio/directpv/pkg/apis/directpv.min.io/v1beta1"
	"github.com/minio/directpv/pkg/clientset"
	typeddirectpv "github.com/minio/directpv/pkg/clientset/typed/directpv.min.io/v1beta1"
)

var Versions = []string{
	"v1beta1",
}

var LatestAddToScheme = directpv.AddToScheme

type (
	Interface = typeddirectpv.DirectpvV1beta1Interface
	Client    = typeddirectpv.DirectpvV1beta1Client

	DriveStatus          = directpv.DriveStatus
	Drive                = directpv.DirectPVDrive
	DriveStatusList      = []directpv.DirectPVDrive
	DriveList            = directpv.DirectPVDriveList
	LatestDriveInterface = typeddirectpv.DirectPVDriveInterface

	VolumeStatus          = directpv.VolumeStatus
	Volume                = directpv.DirectPVVolume
	VolumeStatusList      = []directpv.DirectPVVolume
	VolumeList            = directpv.DirectPVVolumeList
	LatestVolumeInterface = typeddirectpv.DirectPVVolumeInterface

	NodeStatus          = directpv.NodeStatus
	Node                = directpv.DirectPVNode
	Device              = directpv.DirectPVDevice
	NodeStatusList      = []directpv.DirectPVNode
	NodeList            = directpv.DirectPVNodeList
	LatestNodeInterface = typeddirectpv.DirectPVNodeInterface

	InitRequestStatus          = directpv.InitRequestStatus
	InitRequest                = directpv.DirectPVInitRequest
	InitRequestStatusList      = []directpv.DirectPVInitRequest
	InitRequestList            = directpv.DirectPVInitRequestList
	LatestInitRequestInterface = typeddirectpv.DirectPVInitRequestInterface
)

var (
	NewDrive  = directpv.NewDirectPVDrive
	NewVolume = directpv.NewDirectPVVolume
	NewNode   = directpv.NewDirectPVNode
)

type ExtClientsetInterface interface {
	clientset.Interface
	DirectpvLatest() Interface
}

// ExtClientset is extended clientset providing latest DirectPV interface.
type ExtClientset struct {
	*clientset.Clientset
}

// DirectpvLatest retrieves the latest interface.
func (c *ExtClientset) DirectpvLatest() Interface {
	return c.DirectpvV1beta1()
}

// NewExtClientset creates extended clientset.
func NewExtClientset(cs *clientset.Clientset) *ExtClientset {
	return &ExtClientset{cs}
}

func (c *ExtFakeClientset) DirectpvLatest() Interface {
	return c.DirectpvV1beta1()
}
