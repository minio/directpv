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

package consts

const (
	// AppName denotes application/library/plugin/tool name
	AppName = "directpv"

	// AppPrettyName denotes application/library/plugin/tool pretty name
	AppPrettyName = "DirectPV"

	// AppCapsName denotes application/library/plugin/tool name in capital letters.
	AppCapsName = "DIRECTPV"

	// Group denotes group name.
	GroupName = AppName + ".min.io"

	// LatestAPIVersion denotes latest API version of drive/volume.
	LatestAPIVersion = "v1beta1"

	// Identity denotes identity value.
	Identity = AppName + "-min-io"

	// StorageClassName denotes storage class name.
	StorageClassName = Identity

	// ControllerName is the name of the controller.
	ControllerName = AppName + "-controller"

	// DriverName is the driver name.
	DriverName = AppName + "-driver"

	// DriveKind is drive CRD kind.
	DriveKind = AppPrettyName + "Drive"

	// VolumeKind is volume CRD kind.
	VolumeKind = AppPrettyName + "Volume"

	// NodeKind is node CRD kind.
	NodeKind = AppPrettyName + "Node"

	// InitRequestKind denotes the InitRequest CRD kind.
	InitRequestKind = AppPrettyName + "InitRequest"

	// DriveResource is drive CRD resource.
	DriveResource = AppName + "drives"

	// VolumeResource is volume CRD resource.
	VolumeResource = AppName + "volumes"

	// NodeResource is node CRD resource.
	NodeResource = AppName + "nodes"

	// InitRequestResource is initrequest CRD resource.
	InitRequestResource = AppName + "initrequests"

	// AppRootDir is application root directory.
	AppRootDir = "/var/lib/" + AppName

	// UdevDataDir is Udev data directory.
	UdevDataDir = "/run/udev/data"

	// MetricsPort is default metrics port.
	MetricsPort = 10443

	// ReadinessPort is default readiness port.
	ReadinessPort = 30443

	// ReadinessPath is default readiness path.
	ReadinessPath = "/ready"

	// MountRootDir is mount root directory.
	MountRootDir = AppRootDir + "/mnt"

	NodeServerName       = "node-server"
	ControllerServerName = "controller"
	NodeControllerName   = "node-controller"

	LegacyNodeServerName       = "legacy-node-server"
	LegacyControllerServerName = "legacy-controller"

	// TmpFS mount
	TmpMountDir = AppRootDir + "/tmp"
)
