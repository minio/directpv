// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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

import "github.com/minio/directpv/pkg/consts"

const (
	// rbac
	clusterRoleVerbList   = "list"
	clusterRoleVerbUse    = "use"
	clusterRoleVerbGet    = "get"
	clusterRoleVerbWatch  = "watch"
	clusterRoleVerbCreate = "create"
	clusterRoleVerbDelete = "delete"
	clusterRoleVerbUpdate = "update"
	clusterRoleVerbPatch  = "patch"

	// Daemonset
	volumeNameMountpointDir          = "mountpoint-dir"
	volumeNameRegistrationDir        = "registration-dir"
	volumeNamePluginDir              = "plugins-dir"
	volumeNameAppRootDir             = consts.AppName + "-common-root"
	appRootDir                       = consts.AppRootDir + "/"
	nodeDriverRegistrarContainerName = "node-driver-registrar"
	healthZContainerPortName         = "healthz"
	healthZContainerPort             = 9898
	livenessProbeContainerName       = "liveness-probe"
	volumeNameSysDir                 = "sysfs"
	volumePathSysDir                 = consts.SysFSDir
	volumeNameDevDir                 = "devfs"
	volumePathDevDir                 = consts.DevDir
	volumeNameRunUdevData            = "run-udev-data-dir"
	volumePathRunUdevData            = consts.UdevDataDir

	// Deployment
	admissionWebhookSecretName     = "validationwebhookcerts"
	admissionControllerWebhookPort = 20443
	admissionControllerWebhookName = "validatinghook"
	validationControllerName       = consts.AppName + "-validation-controller"
	admissionControllerCertsDir    = "admission-webhook-certs"
	admissionCertsDir              = "/etc/admission/certs"
	csiProvisionerContainerName    = "csi-provisioner"
	admissionWehookDNSName         = consts.AppName + "-validation-controller." + consts.Identity + ".svc"

	// validation rules
	validationWebhookConfigName = "drive.validation.controller"

	// Common
	volumeNameSocketDir       = "socket-dir"
	socketDir                 = "/csi"
	socketFile                = "/csi.sock"
	selectorKey               = "selector." + consts.GroupName
	containerName             = consts.AppName
	kubeNodeNameEnvVarName    = "KUBE_NODE_NAME"
	csiEndpointEnvVarName     = "CSI_ENDPOINT"
	kubeletDirPath            = "/var/lib/kubelet"
	pluginName                = "kubectl-" + consts.AppName
	selectorValueEnabled      = "enabled"
	serviceSelector           = "selector." + consts.GroupName + ".service"
	healthZContainerPortPath  = "/healthz"
	deleteProtectionFinalizer = "/delete-protection"

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

	// readiness
	readinessPortName = "readinessport"
)
