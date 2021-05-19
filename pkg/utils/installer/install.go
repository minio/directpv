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
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/minio/direct-csi/pkg/utils"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	CreatedByLabel      = "created-by"
	DirectCSIPluginName = "kubectl-direct_csi"

	CSIDriver = "CSIDriver"
	DirectCSI = "direct.csi.min.io"

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

	// "csi-provisioner:v2.1.0"
	csiProvisionerContainerImage = "csi-provisioner@sha256:4ca2ce98430ca0b87d5bc1a6d116ecdf1619cfe6db693d8d5aa438f6821e0e80"
	// "livenessprobe:v2.1.0"
	livenessProbeContainerImage = "livenessprobe@sha256:6f056a175ff4ead772edc9bf99aef74c275a83c51868dd26090dcb623425a742"
	// "csi-node-driver-registrar:v2.1.0"
	nodeDriverRegistrarContainerImage = "csi-node-driver-registrar@sha256:9f9ce5c98e44d66b8ad34351616fdf78765b9f24c3c3b496cee784dadf63f528"

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
	conversionWebhookName       = "directcsi-conversion-webhook"
	ConversionWebhookSecretName = "conversionwebhookcerts"
	conversionWebhookPortName   = "convwebhook"
	conversionWebhookPort       = 443

	conversionWebhookCertVolume  = "conversion-webhook-certs"
	conversionWebhookCertsSecret = "converionwebhookcertsecret"
	caCertFileName               = "ca.pem"
	caDir                        = "/etc/CAs"
)

var (
	validationWebhookCaBundle  []byte
	conversionWebhookCaBundle  []byte
	ErrKubeVersionNotSupported = errors.New(
		fmt.Sprintf("%s: This version of kubernetes is not supported by direct-csi. Please upgrade your kubernetes installation and try again", utils.Red("ERR")))
	ErrEmptyCABundle = errors.New("CA bundle is empty")
)

func objMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      sanitizeName(name),
		Namespace: sanitizeName(name),
		Annotations: map[string]string{
			CreatedByLabel: DirectCSIPluginName,
		},
		Labels: map[string]string{
			"app":  DirectCSI,
			"type": CSIDriver,
		},
	}

}

func CreateNamespace(ctx context.Context, identity string, dryRun bool) error {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: objMeta(identity),
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{},
		},
		Status: corev1.NamespaceStatus{},
	}

	if dryRun {
		return utils.LogYAML(ns)
	}

	// Create Namespace Obj
	if _, err := utils.GetKubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateCSIDriver(ctx context.Context, identity string, dryRun bool) error {
	podInfoOnMount := true
	attachRequired := false

	version := "v1"
	if !dryRun {
		gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	switch version {
	case "v1":
		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CSIDriver",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: objMeta(identity),
			Spec: storagev1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
					storagev1.VolumeLifecyclePersistent,
					storagev1.VolumeLifecycleEphemeral,
				},
			},
		}

		if dryRun {
			return utils.LogYAML(csiDriver)
		}

		// Create CSIDriver Obj
		if _, err := utils.GetKubeClient().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CSIDriver",
				APIVersion: "storage.k8s.io/v1beta1",
			},
			ObjectMeta: objMeta(identity),
			Spec: storagev1beta1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1beta1.VolumeLifecycleMode{
					storagev1beta1.VolumeLifecyclePersistent,
					storagev1beta1.VolumeLifecycleEphemeral,
				},
			},
		}

		if dryRun {
			return utils.LogYAML(csiDriver)
		}

		// Create CSIDriver Obj
		if _, err := utils.GetKubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func CreateStorageClass(ctx context.Context, identity string, dryRun bool) error {
	allowExpansion := false
	allowedTopologies := []corev1.TopologySelectorTerm{}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	version := "v1"
	if !dryRun {
		gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	switch version {
	case "v1":
		bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		// Create StorageClass for the new driver
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta:           objMeta(identity),
			Provisioner:          sanitizeName(identity),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies:    allowedTopologies,
			ReclaimPolicy:        &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if dryRun {
			return utils.LogYAML(storageClass)
		}

		if _, err := utils.GetKubeClient().StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		// Create StorageClass for the new driver
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1beta1",
			},
			ObjectMeta:           objMeta(identity),
			Provisioner:          sanitizeName(identity),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies:    allowedTopologies,
			ReclaimPolicy:        &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if dryRun {
			return utils.LogYAML(storageClass)
		}

		if _, err := utils.GetKubeClient().StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func CreateService(ctx context.Context, identity string, dryRun bool) error {
	csiPort := corev1.ServicePort{
		Port: 12345,
		Name: "unused",
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: objMeta(identity),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{csiPort},
			Selector: map[string]string{
				"app":  DirectCSI,
				"type": CSIDriver,
			},
		},
	}

	if dryRun {
		return utils.LogYAML(svc)
	}

	if _, err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func getConversionWebhookDNSName(identity string) string {
	return strings.Join([]string{conversionWebhookName, sanitizeName(identity), "svc"}, ".") // "directcsi-conversion-webhook.direct-csi-min-io.svc"
}

func getConversionWebhookURL(identity string) (conversionWebhookURL string) {
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	conversionWebhookURL = fmt.Sprintf("https://%s", conversionWebhookDNSName+healthZContainerPortPath) // https://directcsi-conversion-webhook.direct-csi-min-io.svc/healthz
	return
}

func CreateDaemonSet(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string, loopBackOnly bool, nodeSelector map[string]string, tolerations []corev1.Toleration) error {
	name := sanitizeName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	conversionWebhookURL := getConversionWebhookURL(identity)

	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		HostNetwork:        false,
		HostIPC:            true,
		HostPID:            true,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, name)),
			newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
			newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
			newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
			newHostPathVolume(volumeNameCSIRootDir, csiRootPath),
			newHostPathVolume(volumeNameSysDir, volumePathSysDir),
			newSecretVolume(conversionWebhookCertVolume, conversionWebhookCertsSecret),
		},
		Containers: []corev1.Container{
			{
				Name:  nodeDriverRegistrarContainerName,
				Image: filepath.Join(registry, org, nodeDriverRegistrarContainerImage),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--csi-address=unix:///csi/csi.sock",
					fmt.Sprintf("--kubelet-registration-path=%s",
						newDirectCSIPluginsSocketDir(kubeletDirPath, name)+"/csi.sock"),
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
					newVolumeMount(volumeNameRegistrationDir, "/registration", false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-registrar-termination-log",
			},
			{
				Name:  directCSIContainerName,
				Image: filepath.Join(registry, org, directCSIContainerImage),
				Args: func() []string {
					args := []string{
						fmt.Sprintf("--identity=%s", name),
						fmt.Sprintf("--v=%d", logLevel),
						fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
						fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
						fmt.Sprintf("--conversion-webhook-url=%s", conversionWebhookURL),
						"--driver",
					}
					if loopBackOnly {
						args = append(args, "--loopback-only")
					}
					return args
				}(),
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
					newVolumeMount(volumeNameMountpointDir, kubeletDirPath+"/pods", true),
					newVolumeMount(volumeNamePluginDir, kubeletDirPath+"/plugins", true),
					newVolumeMount(volumeNameCSIRootDir, csiRootPath, true),
					newVolumeMount(volumeNameSysDir, "/sys", true),
					newVolumeMount(conversionWebhookCertVolume, caDir, false),
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 9898,
						Name:          "healthz",
						Protocol:      corev1.ProtocolTCP,
					},
				},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 10,
					TimeoutSeconds:      3,
					PeriodSeconds:       2,
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: healthZContainerPortPath,
							Port: intstr.FromString(healthZContainerPortName),
						},
					},
				},
			},
			{
				Name:  livenessProbeContainerName,
				Image: filepath.Join(registry, org, livenessProbeContainerImage),
				Args: []string{
					"--csi-address=/csi/csi.sock",
					"--health-port=9898",
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
				},
			},
		},
		NodeSelector: nodeSelector,
		Tolerations:  tolerations,
	}

	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objMeta(identity),
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sanitizeName(name),
					Namespace: sanitizeName(name),
					Annotations: map[string]string{
						CreatedByLabel: DirectCSIPluginName,
					},
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}

	if dryRun {
		return utils.LogYAML(daemonset)
	}

	if _, err := utils.GetKubeClient().AppsV1().DaemonSets(sanitizeName(identity)).Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateControllerService(ctx context.Context, generatedSelectorValue, identity string, dryRun bool) error {
	admissionWebhookPort := corev1.ServicePort{
		Port: admissionControllerWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: admissionControllerWebhookName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      validationControllerName,
			Namespace: sanitizeName(identity),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{admissionWebhookPort},
			Selector: map[string]string{
				directCSISelector: generatedSelectorValue,
			},
		},
	}

	if dryRun {
		return utils.LogYAML(svc)
	}

	if _, err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateControllerSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte, dryRun bool) error {

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[privateKeyFileName] = privateKeyBytes
		mp[publicCertFileName] = publicCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AdmissionWebhookSecretName,
			Namespace: sanitizeName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionCASecret(ctx context.Context, identity string, caCertBytes []byte, dryRun bool) error {

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[caCertFileName] = caCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      conversionWebhookCertsSecret,
			Namespace: sanitizeName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateDeployment(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string) error {
	name := sanitizeName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	conversionWebhookURL := getConversionWebhookURL(identity)

	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", name))),
			newSecretVolume(admissionControllerCertsDir, AdmissionWebhookSecretName),
			newSecretVolume(conversionWebhookCertVolume, conversionWebhookCertsSecret),
		},
		Containers: []corev1.Container{
			{
				Name:  csiProvisionerContainerName,
				Image: filepath.Join(registry, org, csiProvisionerContainerImage),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--timeout=300s",
					fmt.Sprintf("--csi-address=$(%s)", endpointEnvVarCSI),
					"--leader-election",
					"--feature-gates=Topology=true",
					"--strict-topology",
				},
				Env: []corev1.EnvVar{
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/controller-provisioner-termination-log",
				// TODO: Enable this after verification
				// LivenessProbe: &corev1.Probe{
				// 	FailureThreshold:    5,
				// 	InitialDelaySeconds: 10,
				// 	TimeoutSeconds:      3,
				// 	PeriodSeconds:       2,
				// 	Handler: corev1.Handler{
				// 		HTTPGet: &corev1.HTTPGetAction{
				// 			Path: healthZContainerPortPath,
				// 			Port: intstr.FromInt(9898),
				// 		},
				// 	},
				// },
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
			},
			{
				Name:  directCSIContainerName,
				Image: filepath.Join(registry, org, directCSIContainerImage),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					fmt.Sprintf("--identity=%s", name),
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					fmt.Sprintf("--conversion-webhook-url=%s", conversionWebhookURL),
					"--controller",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: admissionControllerWebhookPort,
						Name:          admissionControllerWebhookName,
						Protocol:      corev1.ProtocolTCP,
					},
					{
						ContainerPort: 9898,
						Name:          "healthz",
						Protocol:      corev1.ProtocolTCP,
					},
				},
				Env: []corev1.EnvVar{
					{
						Name: kubeNodeNameEnvVar,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name:  endpointEnvVarCSI,
						Value: "unix:///csi/csi.sock",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", false),
					newVolumeMount(admissionControllerCertsDir, certsDir, false),
					newVolumeMount(conversionWebhookCertVolume, caDir, false),
				},
			},
		},
	}

	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{admissionWehookDNSName})
	if certErr != nil {
		return certErr
	}
	validationWebhookCaBundle = caCertBytes

	if err := CreateControllerSecret(ctx, identity, publicCertBytes, privateKeyBytes, dryRun); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objMeta(identity),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sanitizeName(name),
					Namespace: sanitizeName(name),
					Annotations: map[string]string{
						CreatedByLabel: DirectCSIPluginName,
					},
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	deployment.ObjectMeta.Finalizers = []string{
		sanitizeName(identity) + DirectCSIFinalizerDeleteProtection,
	}

	if dryRun {
		return utils.LogYAML(deployment)
	}

	if _, err := utils.GetKubeClient().AppsV1().Deployments(sanitizeName(identity)).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if err := CreateControllerService(ctx, generatedSelectorValue, identity, dryRun); err != nil {
		return err
	}

	return nil
}

func sanitizeName(s string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	s = re.ReplaceAllString(s, "-")
	if s[len(s)-1] == '-' {
		s = s + "X"
	}
	return s
}

// Exported
func SanitizeName(s string) string {
	return sanitizeName(s)
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := sanitizeName(name)
	// Max length of name is 255. If needed, cut out last 6 bytes
	// to make room for randomstring
	if len(sanitizedName) >= 255 {
		sanitizedName = sanitizedName[0:249]
	}

	// Get a 5 byte randomstring
	shortUUID := NewRandomString(5)

	// Concatenate sanitizedName (249) and shortUUID (5) with a '-' in between
	// Max length of the returned name cannot be more than 255 bytes
	return fmt.Sprintf("%s-%s", sanitizedName, shortUUID)
}

func newHostPathVolume(name, path string) corev1.Volume {
	hostPathType := corev1.HostPathDirectoryOrCreate
	volumeSource := corev1.VolumeSource{
		HostPath: &corev1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		},
	}

	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newSecretVolume(name, secretName string) corev1.Volume {
	volumeSource := corev1.VolumeSource{
		Secret: &corev1.SecretVolumeSource{
			SecretName: secretName,
		},
	}
	return corev1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	}
}

func newDirectCSIPluginsSocketDir(kubeletDir, name string) string {
	return filepath.Join(kubeletDir, "plugins", sanitizeName(name))
}

func newVolumeMount(name, path string, bidirectional bool) corev1.VolumeMount {
	mountProp := corev1.MountPropagationNone
	if bidirectional {
		mountProp = corev1.MountPropagationBidirectional
	}
	return corev1.VolumeMount{
		Name:             name,
		MountPath:        path,
		MountPropagation: &mountProp,
	}
}

func getDriveValidatingWebhookConfig(identity string) admissionv1.ValidatingWebhookConfiguration {

	name := sanitizeName(identity)
	getServiceRef := func() *admissionv1.ServiceReference {
		path := "/validatedrive"
		return &admissionv1.ServiceReference{
			Namespace: name,
			Name:      validationControllerName,
			Path:      &path,
		}
	}

	getClientConfig := func() admissionv1.WebhookClientConfig {
		return admissionv1.WebhookClientConfig{
			Service:  getServiceRef(),
			CABundle: []byte(validationWebhookCaBundle),
		}

	}

	getValidationRules := func() []admissionv1.RuleWithOperations {
		return []admissionv1.RuleWithOperations{
			{
				Operations: []admissionv1.OperationType{admissionv1.Update},
				Rule: admissionv1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"*"},
					Resources:   []string{"directcsidrives"},
				},
			},
		}
	}

	getValidatingWebhooks := func() []admissionv1.ValidatingWebhook {
		supportedReviewVersions := []string{"v1", "v1beta1"}
		sideEffectClass := admissionv1.SideEffectClassNone
		return []admissionv1.ValidatingWebhook{
			{
				Name:                    ValidationWebhookConfigName,
				ClientConfig:            getClientConfig(),
				AdmissionReviewVersions: supportedReviewVersions,
				SideEffects:             &sideEffectClass,
				Rules:                   getValidationRules(),
			},
		}
	}

	validatingWebhookConfiguration := admissionv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ValidationWebhookConfigName,
			Namespace: name,
			Finalizers: []string{
				sanitizeName(identity) + DirectCSIFinalizerDeleteProtection,
			},
		},
		Webhooks: getValidatingWebhooks(),
	}

	return validatingWebhookConfiguration
}

func RegisterDriveValidationRules(ctx context.Context, identity string, dryRun bool) error {
	driveValidatingWebhookConfig := getDriveValidatingWebhookConfig(identity)
	if dryRun {
		return utils.LogYAML(driveValidatingWebhookConfig)
	}

	if _, err := utils.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, &driveValidatingWebhookConfig, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte, dryRun bool) error {

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[privateKeyFileName] = privateKeyBytes
		mp[publicCertFileName] = publicCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConversionWebhookSecretName,
			Namespace: sanitizeName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionService(ctx context.Context, generatedSelectorValue, identity string, dryRun bool) error {
	webhookPort := corev1.ServicePort{
		Port: conversionWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: conversionWebhookPortName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      conversionWebhookName,
			Namespace: sanitizeName(identity),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{webhookPort},
			Selector: map[string]string{
				directCSISelector: generatedSelectorValue,
			},
		},
	}

	if dryRun {
		return utils.LogYAML(svc)
	}

	if _, err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionDeployment(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string) error {
	name := sanitizeName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newSecretVolume(conversionWebhookName, ConversionWebhookSecretName),
		},
		Containers: []corev1.Container{
			{
				Name:  directCSIContainerName,
				Image: filepath.Join(registry, org, directCSIContainerImage),
				Args: []string{
					"--conversion-webhook",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: conversionWebhookPort,
						Name:          conversionWebhookPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				// // Enable after investigating the CrashLoopBackOff on pods
				// LivenessProbe: &corev1.Probe{
				// 	FailureThreshold:    5,
				// 	InitialDelaySeconds: 10,
				// 	TimeoutSeconds:      3,
				// 	PeriodSeconds:       2,
				// 	Handler: corev1.Handler{
				// 		HTTPGet: &corev1.HTTPGetAction{
				// 			Path:   healthZContainerPortPath,
				// 			Port:   intstr.FromString(conversionWebhookPortName),
				// 			Host:   conversionWebhookDNSName,
				// 			Scheme: corev1.URISchemeHTTPS,
				// 		},
				// 	},
				// },
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(conversionWebhookName, certsDir, false),
				},
			},
		},
	}

	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{conversionWebhookDNSName})
	if certErr != nil {
		return certErr
	}
	conversionWebhookCaBundle = caCertBytes

	if err := CreateConversionSecret(ctx, identity, publicCertBytes, privateKeyBytes, dryRun); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	if err := CreateConversionCASecret(ctx, identity, caCertBytes, dryRun); err != nil {
		if !kerr.IsAlreadyExists(err) {
			return err
		}
	}

	getObjMeta := func() metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name:      conversionWebhookName,
			Namespace: sanitizeName(name),
			Annotations: map[string]string{
				CreatedByLabel: DirectCSIPluginName,
			},
			Labels: map[string]string{
				"app": DirectCSI,
			},
		}
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: getObjMeta(),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      conversionWebhookName,
					Namespace: sanitizeName(name),
					Annotations: map[string]string{
						CreatedByLabel: DirectCSIPluginName,
					},
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	deployment.ObjectMeta.Finalizers = []string{
		sanitizeName(identity) + DirectCSIFinalizerDeleteProtection,
	}

	if dryRun {
		utils.LogYAML(deployment)
	} else {
		if _, err := utils.GetKubeClient().AppsV1().Deployments(sanitizeName(identity)).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	if err := CreateConversionService(ctx, generatedSelectorValue, identity, dryRun); err != nil {
		return err
	}

	return nil
}

func GetConversionCABundle() ([]byte, error) {
	if len(conversionWebhookCaBundle) == 0 {
		return []byte{}, ErrEmptyCABundle
	}
	return conversionWebhookCaBundle, nil
}

func GetConversionServiceName() string {
	return conversionWebhookName
}
