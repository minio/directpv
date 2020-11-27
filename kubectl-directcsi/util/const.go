/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package util

const (
	// Label to denote the creator
	createByLabel = "created-by"

	// Denotes that it was created by direct-csi-controller
	directCSIController = "controller.direct.csi.min.io"

	clusterRoleVerbList   = "list"
	clusterRoleVerbGet    = "get"
	clusterRoleVerbWatch  = "watch"
	clusterRoleVerbCreate = "create"
	clusterRoleVerbDelete = "delete"
	clusterRoleVerbUpdate = "update"
	clusterRoleVerbPatch  = "patch"

	volumeNameSocketDir       = "socket-dir"
	volumeNameDevDir          = "dev-dir"
	volumePathDevDir          = "/dev"
	volumeNameSysDir          = "sys-fs"
	volumePathSysDir          = "/sys"
	volumeNameProcDir         = "proc-fs"
	volumePathProcDir         = "/proc"
	volumeNameCSIRootDir      = "direct-csi-common-root"
	volumeNameMountpointDir   = "mountpoint-dir"
	volumeNameRegistrationDir = "registration-dir"
	volumeNamePluginDir       = "plugins-dir"

	directCSISelector = "selector.direct.csi.min.io"

	csiProvisionerContainerName  = "csi-provisioner"
	csiProvisionerContainerImage = "quay.io/k8scsi/csi-provisioner:v1.2.1"

	directCSIContainerName  = "direct-csi"
	directCSIContainerImage = "minio/direct-csi:v0.2.1"

	livenessProbeContainerName  = "liveness-probe"
	livenessProbeContainerImage = "quay.io/k8scsi/livenessprobe:v1.1.0"

	nodeDriverRegistrarContainerName  = "node-driver-registrar"
	nodeDriverRegistrarContainerImage = "quay.io/k8scsi/csi-node-driver-registrar:v2.0.0"

	healthZContainerPort         = 9898
	healthZContainerPortName     = "healthz"
	healthZContainerPortProtocol = "TCP"
	healthZContainerPortPath     = "/healthz"

	kubeNodeNameEnvVar = "KUBE_NODE_NAME"
	endpointEnvVarCSI  = "CSI_ENDPOINT"
)
