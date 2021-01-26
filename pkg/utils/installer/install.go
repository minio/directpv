// This file is part of MinIO Direct CSI
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

package installer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/minio/direct-csi/pkg/utils"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kErr "k8s.io/apimachinery/pkg/api/errors"
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

	csiProvisionerContainerName  = "csi-provisioner"
	csiProvisionerContainerImage = "quay.io/k8scsi/csi-provisioner:v1.2.1"

	directCSIContainerName = "direct-csi"

	livenessProbeContainerName  = "liveness-probe"
	livenessProbeContainerImage = "quay.io/k8scsi/livenessprobe:v1.1.0"

	nodeDriverRegistrarContainerName  = "node-driver-registrar"
	nodeDriverRegistrarContainerImage = "quay.io/k8scsi/csi-node-driver-registrar:v1.3.0"

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
)

var (
	caBundle                   []byte
	ErrKubeVersionNotSupported = errors.New(
		fmt.Sprintf("%s: This version of kubernetes is not supported by direct-csi. Please upgrade your kubernetes installation and try again", utils.Red("ERR")))
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

func CreateNamespace(ctx context.Context, identity string) error {
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

	// Create Namespace Obj
	if _, err := utils.GetKubeClient().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateCSIDriver(ctx context.Context, identity string) error {
	podInfoOnMount := true
	attachRequired := false

	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
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

		// Create CSIDriver Obj
		if _, err := utils.GetKubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func CreateStorageClass(ctx context.Context, identity string) error {
	allowExpansion := false
	allowedTopologies := []corev1.TopologySelectorTerm{}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
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
			Provisioner:          identity,
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies:    allowedTopologies,
			ReclaimPolicy:        &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if _, err := utils.GetKubeClient().StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func CreateService(ctx context.Context, identity string) error {
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

	if _, err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateDaemonSet(ctx context.Context, identity string, directCSIContainerImage string) error {
	name := sanitizeName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)

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
		},
		Containers: []corev1.Container{
			{
				Name:  nodeDriverRegistrarContainerName,
				Image: nodeDriverRegistrarContainerImage,
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
				Image: directCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--identity=%s", name),
					fmt.Sprintf("--v=%d", logLevel),
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
					"--driver",
				},
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
				Image: livenessProbeContainerImage,
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
	if _, err := utils.GetKubeClient().AppsV1().DaemonSets(sanitizeName(identity)).Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateControllerService(ctx context.Context, generatedSelectorValue, identity string) error {
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

	if _, err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateControllerSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte) error {

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

	if _, err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateDeployment(ctx context.Context, identity string, directCSIContainerImage string) error {
	name := sanitizeName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)

	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", name))),
			newSecretVolume(admissionControllerCertsDir, AdmissionWebhookSecretName),
		},
		Containers: []corev1.Container{
			{
				Name:  csiProvisionerContainerName,
				Image: csiProvisionerContainerImage,
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--timeout=300s",
					fmt.Sprintf("--csi-address=$(%s)", endpointEnvVarCSI),
					"--enable-leader-election",
					"--leader-election-type=leases",
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
				Image: directCSIContainerImage,
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					fmt.Sprintf("--identity=%s", name),
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
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
				},
			},
		},
	}

	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{admissionWehookDNSName})
	if certErr != nil {
		return certErr
	}
	caBundle = caCertBytes

	if err := CreateControllerSecret(ctx, identity, publicCertBytes, privateKeyBytes); err != nil {
		if !kErr.IsAlreadyExists(err) {
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

	if _, err := utils.GetKubeClient().AppsV1().Deployments(sanitizeName(identity)).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if err := CreateControllerService(ctx, generatedSelectorValue, identity); err != nil {
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
			CABundle: []byte(caBundle),
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

func RegisterDriveValidationRules(ctx context.Context, identity string) error {
	driveValidatingWebhookConfig := getDriveValidatingWebhookConfig(identity)
	if _, err := utils.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, &driveValidatingWebhookConfig, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
