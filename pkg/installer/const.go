// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package installer

import (
	"time"
)

// CSI provisioner images
const (
	// quay.io/minio/csi-provisioner:v2.2.2
	CSIImageCSIProvisioner = "csi-provisioner@sha256:3b465cbcadf7d437fc70c3b6aa2c93603a7eef0a3f5f1e861d91f303e4aabdee"

	// quay.io/minio/csi-node-driver-registrar:v2.2.0
	CSIImageNodeDriverRegistrar = "csi-node-driver-registrar@sha256:ba763bb01ddc09e312240c8abc310aa2e2dd6aee636d342f6dd9238a6bff179c"

	// quay.io/minio/livenessprobe:v2.2.0
	CSIImageLivenessProbe = "livenessprobe@sha256:072e29e350ed7e870e119cbba37324348e1d00f0ba06d4ea288413466d1aa8e8"
)

// Misc
const (
	CreatedByLabel      = "created-by"
	DirectCSIPluginName = "kubectl/direct-csi"

	AppNameLabel = "application-name"
	AppTypeLabel = "application-type"

	CSIDriver = "CSIDriver"
	DirectCSI = "direct.csi.min.io"
)

const (
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
	volumeNameCSIRootDir      = "direct-csi-common-root"
	volumeNameMountpointDir   = "mountpoint-dir"
	volumeNameRegistrationDir = "registration-dir"
	volumeNamePluginDir       = "plugins-dir"

	directCSISelector = "selector.direct.csi.min.io"

	directCSIContainerName           = "direct-csi"
	livenessProbeContainerName       = "liveness-probe"
	nodeDriverRegistrarContainerName = "node-driver-registrar"
	csiProvisionerContainerName      = "csi-provisioner"

	// "csi-provisioner:v2.2.0"
	csiProvisionerContainerImage = "csi-provisioner@sha256:3b465cbcadf7d437fc70c3b6aa2c93603a7eef0a3f5f1e861d91f303e4aabdee"
	// "livenessprobe:v2.2.0"
	livenessProbeContainerImage = "livenessprobe@sha256:072e29e350ed7e870e119cbba37324348e1d00f0ba06d4ea288413466d1aa8e8"
	// "csi-node-driver-registrar:v2.2.0"
	nodeDriverRegistrarContainerImage = "csi-node-driver-registrar@sha256:ba763bb01ddc09e312240c8abc310aa2e2dd6aee636d342f6dd9238a6bff179c"

	healthZContainerPort         = 9898
	healthZContainerPortName     = "healthz"
	healthZContainerPortProtocol = "TCP"
	healthZContainerPortPath     = "/healthz"

	kubeNodeNameEnvVar = "KUBE_NODE_NAME"
	endpointEnvVarCSI  = "CSI_ENDPOINT"

	kubeletDirPath = "/var/lib/kubelet"
	csiRootPath    = "/var/lib/direct-csi/"

	// debug log level default
	logLevel = 3

	// Admission controller
	admissionControllerCertsDir    = "admission-webhook-certs"
	AdmissionWebhookSecretName     = "validationwebhookcerts"
	validationControllerName       = "directcsi-validation-controller"
	admissionControllerWebhookName = "validatinghook"
	ValidationWebhookConfigName    = "drive.validation.controller"
	admissionControllerWebhookPort = 443
	certsDir                       = "/etc/certs"
	admissionWehookDNSName         = "directcsi-validation-controller.direct-csi-min-io.svc"
	privateKeyFileName             = "key.pem"
	publicCertFileName             = "cert.pem"

	// Finalizers
	DirectCSIFinalizerDeleteProtection = "/delete-protection"

	// Conversion webhook
	conversionWebhookName                  = "directcsi-conversion-webhook"
	ConversionWebhookSecretName            = "conversionwebhookcerts"
	conversionWebhookPortName              = "convwebhook"
	conversionWebhookPort                  = 443
	conversionDeploymentReadinessThreshold = 2
	conversionDeploymentRetryInterval      = 3 * time.Second

	conversionWebhookCertVolume  = "conversion-webhook-certs"
	conversionWebhookCertsSecret = "converionwebhookcertsecret"
	caCertFileName               = "ca.pem"
	caDir                        = "/etc/CAs"
)
