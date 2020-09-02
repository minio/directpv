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

package centralcontroller

const (
	// The namespace in which direct-csi related resources will be created
	DirectCSINS = "direct-csi-min-io"

	// Label to denote the creator
	CreateByLabel = "created-by"

	// Denotes that it was created by direct-csi-controller
	DirectCSIController = "controller.direct.csi.min.io"

	// Selector key for matching pods to corresponding pod controllers (DaemonSet, Deployment etc.)
	DirectCSISelector = "selector.direct.csi.min.io"

	// VolumeNameSocketDir denotes the name of the volume in the pod spec for the socket directory
	VolumeNameSocketDir = "socket-dir"

	// Kubelet directory for plugins
	KubeletDir = "/var/lib/kubelet"

	// Name of the volume for mountpoint-dir
	VolumeNameMountpointDir = "mountpoint-dir"

	// The directory of the pod mountpoint. Must be used with the KubeletDir as the containing directory
	VolumePathMountpointDir = "pods"

	VolumeNamePluginsDir = "plugins"
	VolumePathPluginsDir = "plugins"

	VolumeNamePluginsRegistryDir = "plugins-registry"
	VolumePathPluginsRegistryDir = "plugins_registry"

	VolumeNameDevDir = "dev"
	VolumePathDevDir = "/dev"

	KubeNodeNameEnvVar = "KUBE_NODE_NAME"
	CSIEndpointEnvVar  = "CSI_ENDPOINT"

	NodeDriverRegistrarContainerName  = "node-driver-registrar"
	NodeDriverRegistrarContainerImage = "quay.io/k8scsi/csi-node-driver-registrar:v1.3.0"

	CSIProvisionerContainerName  = "csi-provisioner"
	CSIProvisionerContainerImage = "quay.io/k8scsi/csi-provisioner:v1.2.1"

	DirectCSIContainerName  = "direct-csi"
	DirectCSIContainerImage = "minio/direct-csi:v0.2.1"

	LivenessProbeContainerName  = "liveness-probe"
	LivenessProbeContainerImage = "quay.io/k8scsi/livenessprobe:v1.1.0"

	HealthZContainerPort         = 9898
	HealthZContainerPortName     = "healthz"
	HealthZContainerPortProtocol = "TCP"
	HealthZContainerPortPath     = "/healthz"

	ClusterRoleVerbGet    = "get"
	ClusterRoleVerbList   = "list"
	ClusterRoleVerbWatch  = "watch"
	ClusterRoleVerbCreate = "create"
	ClusterRoleVerbDelete = "delete"
	ClusterRoleVerbUpdate = "update"
	ClusterRoleVerbPatch  = "patch"
)
