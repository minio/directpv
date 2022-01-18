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

const (
	// conversion deployment
	conversionWebhookDeploymentName = "directcsi-conversion-webhook"
	conversionWebhookSecretName     = "conversionwebhookcerts"
	conversionWebhookCertsSecret    = "converionwebhookcertsecret"

	// rbac
	clusterRoleVerbList   = "list"
	clusterRoleVerbGet    = "get"
	clusterRoleVerbWatch  = "watch"
	clusterRoleVerbCreate = "create"
	clusterRoleVerbDelete = "delete"
	clusterRoleVerbUpdate = "update"
	clusterRoleVerbPatch  = "patch"

	// conversion secret
	conversionKeyPair = "conversionkeypair"
	caCertFileName    = "ca.pem"
	conversionCACert  = "conversioncacert"

	// crd
	driveCRDName  = "directcsidrives.direct.csi.min.io"
	volumeCRDName = "directcsivolumes.direct.csi.min.io"

	// Daemonset
	volumeNameMountpointDir          = "mountpoint-dir"
	volumeNameRegistrationDir        = "registration-dir"
	volumeNamePluginDir              = "plugins-dir"
	volumeNameCSIRootDir             = "direct-csi-common-root"
	csiRootPath                      = "/var/lib/direct-csi/"
	nodeDriverRegistrarContainerName = "node-driver-registrar"
	healthZContainerPortName         = "healthz"
	livenessProbeContainerName       = "liveness-probe"
	volumeNameSysDir                 = "sysfs"
	volumePathSysDir                 = "/sys"
	volumeNameDevDir                 = "devfs"
	volumePathDevDir                 = "/dev"
	volumeNameRunUdevData            = "run-udev-data-dir"
	volumePathRunUdevData            = "/run/udev/data"

	// Deployment
	admissionWebhookSecretName     = "validationwebhookcerts"
	admissionControllerWebhookPort = 20443
	admissionControllerWebhookName = "validatinghook"
	validationControllerName       = "directcsi-validation-controller"
	admissionControllerCertsDir    = "admission-webhook-certs"
	admissionCertsDir              = "/etc/admission/certs"
	csiProvisionerContainerName    = "csi-provisioner"
	admissionWehookDNSName         = "directcsi-validation-controller.direct-csi-min-io.svc"

	// validation rules
	validationWebhookConfigName = "drive.validation.controller"

	// Common
	volumeNameSocketDir                = "socket-dir"
	directCSISelector                  = "selector.direct.csi.min.io"
	directCSIContainerName             = "direct-csi"
	kubeNodeNameEnvVar                 = "KUBE_NODE_NAME"
	endpointEnvVarCSI                  = "CSI_ENDPOINT"
	kubeletDirPath                     = "/var/lib/kubelet"
	directCSIPluginName                = "kubectl-direct-csi"
	conversionWebhookPortName          = "convwebhook"
	conversionWebhookPort              = 30443
	selectorValueEnabled               = "enabled"
	conversionCADir                    = "/etc/conversion/CAs"
	conversionCertsDir                 = "/etc/conversion/certs"
	webhookSelector                    = "selector.direct.csi.min.io.webhook"
	healthZContainerPortPath           = "/healthz"
	directCSIFinalizerDeleteProtection = "/delete-protection"

	// debug log level default
	logLevel = 3

	// key-pairs
	privateKeyFileName = "key.pem"
	publicCertFileName = "cert.pem"

	// string-gen
	charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	// Misc
	createdByLabel = "created-by"
	appNameLabel   = "application-name"
	appTypeLabel   = "application-type"

	// metrics
	metricsPortName = "metrics"
	metricsPort     = 10443
)
