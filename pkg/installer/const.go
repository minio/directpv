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

// CSI provisioner images
const (
	// quay.io/minio/csi-provisioner:v2.2.0-go1.17
	CSIImageCSIProvisioner = "csi-provisioner@sha256:d4f94539565cf62aea57062b6a42c5156337003133fd3f51b93df9a789e69840"

	// quay.io/minio/csi-node-driver-registrar:v2.2.0-go1.17
	CSIImageNodeDriverRegistrar = "csi-node-driver-registrar@sha256:843fb23b1a3fa1de986378b0b8c08c35f8e62499d386de8ec57801fd029afe6d"

	// quay.io/minio/livenessprobe:v2.2.0-go1.17
	CSIImageLivenessProbe = "livenessprobe@sha256:928a80be4d363e0e438ff28dcdb00d8d674d3059c6149a8cda64ce6016a9a3f8"
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
	selectorValueEnabled  = "enabled"

	volumeNameSocketDir       = "socket-dir"
	volumeNameSysDir          = "sys-fs"
	volumePathSysDir          = "/sys"
	volumeNameRunUdevData     = "run-udev-data"
	volumePathRunUdevData     = "/run/udev/data"
	volumeNameCSIRootDir      = "direct-csi-common-root"
	volumeNameMountpointDir   = "mountpoint-dir"
	volumeNameRegistrationDir = "registration-dir"
	volumeNamePluginDir       = "plugins-dir"

	directCSISelector = "selector.direct.csi.min.io"

	directCSIContainerName           = "direct-csi"
	livenessProbeContainerName       = "liveness-probe"
	nodeDriverRegistrarContainerName = "node-driver-registrar"
	csiProvisionerContainerName      = "csi-provisioner"

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
	admissionControllerWebhookPort = 20443
	admissionCertsDir              = "/etc/admission/certs"
	admissionWehookDNSName         = "directcsi-validation-controller.direct-csi-min-io.svc"
	privateKeyFileName             = "key.pem"
	publicCertFileName             = "cert.pem"

	// Finalizers
	DirectCSIFinalizerDeleteProtection = "/delete-protection"

	// Conversion webhook (Legacy)
	conversionWebhookDeploymentName = "directcsi-conversion-webhook"
	ConversionWebhookSecretName     = "conversionwebhookcerts"
	conversionWebhookCertsSecret    = "converionwebhookcertsecret"

	// conversion
	conversionKeyPair         = "conversionkeypair"
	conversionCACert          = "conversioncacert"
	conversionWebhookPortName = "convwebhook"
	ConversionWebhookPort     = 30443
	caCertFileName            = "ca.pem"
	conversionCADir           = "/etc/conversion/CAs"
	conversionCertsDir        = "/etc/conversion/certs"
	webhookSelector           = "selector.direct.csi.min.io.webhook"
)
