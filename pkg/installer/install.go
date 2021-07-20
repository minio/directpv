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
	"strings"
	"time"

	"github.com/minio/direct-csi/pkg/topology"
	"github.com/minio/direct-csi/pkg/utils"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

var (
	validationWebhookCaBundle []byte
	conversionWebhookCaBundle []byte

	ErrKubeVersionNotSupported = errors.New(
		utils.Red("Error") +
			"This version of kubernetes is not supported by direct-csi" +
			"Please upgrade your kubernetes installation and try again",
	)
	ErrEmptyCABundle = errors.New("CA bundle is empty")
)

func objMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      utils.SanitizeKubeResourceName(name),
		Namespace: utils.SanitizeKubeResourceName(name),
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

	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	version := gvk.Version

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

func getTopologySelectorTerm(identity string) corev1.TopologySelectorTerm {

	getIdentityLabelRequirement := func() corev1.TopologySelectorLabelRequirement {
		return corev1.TopologySelectorLabelRequirement{
			Key:    topology.TopologyDriverIdentity,
			Values: []string{utils.SanitizeKubeResourceName(identity)},
		}
	}

	return corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			getIdentityLabelRequirement(),
		},
	}
}

func CreateStorageClass(ctx context.Context, identity string, dryRun bool) error {
	allowExpansion := false
	allowedTopologies := []corev1.TopologySelectorTerm{
		getTopologySelectorTerm(identity),
	}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	version := gvk.Version

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
			Provisioner:          utils.SanitizeKubeResourceName(identity),
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
			Provisioner:          utils.SanitizeKubeResourceName(identity),
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

	if _, err := utils.GetKubeClient().CoreV1().Services(utils.SanitizeKubeResourceName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func getConversionWebhookDNSName(identity string) string {
	return strings.Join([]string{conversionWebhookName, utils.SanitizeKubeResourceName(identity), "svc"}, ".") // "directcsi-conversion-webhook.direct-csi-min-io.svc"
}

func getConversionWebhookURL(identity string) (conversionWebhookURL string) {
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	conversionWebhookURL = fmt.Sprintf("https://%s", conversionWebhookDNSName+healthZContainerPortPath) // https://directcsi-conversion-webhook.direct-csi-min-io.svc/healthz
	return
}

func CreateDaemonSet(ctx context.Context,
	identity string,
	directCSIContainerImage string,
	dryRun bool,
	registry, org string,
	loopBackOnly bool,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
	seccompProfileName, apparmorProfileName string) error {

	name := utils.SanitizeKubeResourceName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	conversionWebhookURL := getConversionWebhookURL(identity)

	privileged := true
	securityContext := &corev1.SecurityContext{Privileged: &privileged}

	if seccompProfileName != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &seccompProfileName,
		}
	}

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
						fmt.Sprintf("-v=%d", logLevel),
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
				SecurityContext: securityContext,
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
					InitialDelaySeconds: 300,
					TimeoutSeconds:      5,
					PeriodSeconds:       5,
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

	annotations := map[string]string{
		CreatedByLabel: DirectCSIPluginName,
	}
	if apparmorProfileName != "" {
		annotations["container.apparmor.security.beta.kubernetes.io/direct-csi"] = apparmorProfileName
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
					Name:        utils.SanitizeKubeResourceName(name),
					Namespace:   utils.SanitizeKubeResourceName(name),
					Annotations: annotations,
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

	if _, err := utils.GetKubeClient().AppsV1().DaemonSets(utils.SanitizeKubeResourceName(identity)).Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
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
			Namespace: utils.SanitizeKubeResourceName(identity),
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

	if _, err := utils.GetKubeClient().CoreV1().Services(utils.SanitizeKubeResourceName(identity)).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
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
			Namespace: utils.SanitizeKubeResourceName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateOrUpdateConversionCASecret(ctx context.Context, identity string, caCertBytes []byte, dryRun bool) error {

	secretsClient := utils.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity))

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
			Namespace: utils.SanitizeKubeResourceName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	existingSecret, err := secretsClient.Get(ctx, conversionWebhookCertsSecret, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func CreateDeployment(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string) error {
	name := utils.SanitizeKubeResourceName(identity)
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
					fmt.Sprintf("-v=%d", logLevel),
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
					Name:      utils.SanitizeKubeResourceName(name),
					Namespace: utils.SanitizeKubeResourceName(name),
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
		utils.SanitizeKubeResourceName(identity) + DirectCSIFinalizerDeleteProtection,
	}

	if dryRun {
		return utils.LogYAML(deployment)
	}

	if _, err := utils.GetKubeClient().AppsV1().Deployments(utils.SanitizeKubeResourceName(identity)).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	if err := CreateControllerService(ctx, generatedSelectorValue, identity, dryRun); err != nil {
		return err
	}

	return nil
}

func generateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := utils.SanitizeKubeResourceName(name)
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
	return filepath.Join(kubeletDir, "plugins", utils.SanitizeKubeResourceName(name))
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

	name := utils.SanitizeKubeResourceName(identity)
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
		supportedReviewVersions := []string{"v1", "v1beta1", "v1beta2"}
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
				utils.SanitizeKubeResourceName(identity) + DirectCSIFinalizerDeleteProtection,
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

	if _, err := utils.GetKubeClient().
		AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Create(ctx, &driveValidatingWebhookConfig, metav1.CreateOptions{}); err != nil {

		return err
	}
	return nil
}

func CreateOrUpdateConversionSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte, dryRun bool) error {

	secretsClient := utils.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity))

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
			Namespace: utils.SanitizeKubeResourceName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	existingSecret, err := secretsClient.Get(ctx, ConversionWebhookSecretName, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
		if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}

	existingSecret.Data = secret.Data
	if _, err := secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func CreateOrUpdateConversionService(ctx context.Context, generatedSelectorValue, identity string, dryRun bool) error {

	servicesClient := utils.GetKubeClient().CoreV1().Services(utils.SanitizeKubeResourceName(identity))
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
			Namespace: utils.SanitizeKubeResourceName(identity),
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

	existingService, err := servicesClient.Get(ctx, conversionWebhookName, metav1.GetOptions{})
	if err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
		if _, err := servicesClient.Create(ctx, svc, metav1.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}

	existingService.Spec.Selector = svc.Spec.Selector
	if _, err := servicesClient.Update(ctx, existingService, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func CreateConversionDeployment(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string) error {
	name := utils.SanitizeKubeResourceName(identity)
	generatedSelectorValue := generateSanitizedUniqueNameFrom(name)
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	var replicas int32 = 3
	privileged := true

	healthZHandler := corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   healthZContainerPortPath,
			Port:   intstr.FromString(conversionWebhookPortName),
			Scheme: corev1.URISchemeHTTPS,
		},
	}

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
				LivenessProbe: &corev1.Probe{
					Handler: healthZHandler,
				},
				ReadinessProbe: &corev1.Probe{
					Handler: healthZHandler,
				},
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

	if err := CreateOrUpdateConversionSecret(ctx, identity, publicCertBytes, privateKeyBytes, dryRun); err != nil {
		return err
	}

	if err := CreateOrUpdateConversionCASecret(ctx, identity, caCertBytes, dryRun); err != nil {
		return err
	}

	getObjMeta := func() metav1.ObjectMeta {
		return metav1.ObjectMeta{
			Name:      conversionWebhookName,
			Namespace: utils.SanitizeKubeResourceName(name),
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
					Namespace: utils.SanitizeKubeResourceName(name),
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
		utils.SanitizeKubeResourceName(identity) + DirectCSIFinalizerDeleteProtection,
	}

	if err := CreateOrUpdateConversionService(ctx, generatedSelectorValue, identity, dryRun); err != nil {
		return err
	}

	if dryRun {
		utils.LogYAML(deployment)
	} else {
		if _, err := utils.GetKubeClient().
			AppsV1().
			Deployments(utils.SanitizeKubeResourceName(identity)).
			Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func GetConversionCABundle(ctx context.Context, identity string, dryRun bool) ([]byte, error) {
	getCABundlerFromGlobal := func() ([]byte, error) {
		if len(conversionWebhookCaBundle) == 0 {
			return []byte{}, ErrEmptyCABundle
		}
		return conversionWebhookCaBundle, nil
	}

	secret, err := utils.GetKubeClient().
		CoreV1().
		Secrets(utils.SanitizeKubeResourceName(identity)).
		Get(ctx, conversionWebhookCertsSecret, metav1.GetOptions{})
	if err != nil {
		if kerr.IsNotFound(err) && dryRun {
			return getCABundlerFromGlobal()
		}
		return []byte{}, err
	}

	for key, value := range secret.Data {
		if key == caCertFileName {
			return value, nil
		}
	}

	return []byte{}, ErrEmptyCABundle
}

func GetConversionServiceName() string {
	return conversionWebhookName
}

func CreateOrUpdateConversionDeployment(ctx context.Context, identity string, directCSIContainerImage string, dryRun bool, registry, org string) error {
	deploymentsClient := utils.GetKubeClient().
		AppsV1().Deployments(utils.SanitizeKubeResourceName(identity))

	deployment, getErr := deploymentsClient.Get(ctx, conversionWebhookName, metav1.GetOptions{})
	if getErr != nil {
		if !kerr.IsNotFound(getErr) {
			return getErr
		}
		if err := CreateConversionDeployment(ctx, identity, directCSIContainerImage, dryRun, registry, org); err != nil {
			return err
		}
		return nil
	}
	// Conversion deployment is already present. Just update the container version.
	deploymentImage := filepath.Join(registry, org, directCSIContainerImage)
	if deployment.Spec.Template.Spec.Containers[0].Image != deploymentImage {
		deployment.Spec.Template.Spec.Containers[0].Image = deploymentImage
		if dryRun {
			deployment.TypeMeta.Kind = "Deployment"
			deployment.TypeMeta.APIVersion = "apps/v1"
			utils.LogYAML(deployment)
		} else {
			if _, err := deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
				return err
			}
			klog.V(5).Infof("Updated the conversion deployment image to: %v", deployment.Spec.Template.Spec.Containers[0].Image)
		}
	}
	return nil
}

func WaitForConversionDeployment(ctx context.Context, identity string) {
	for {
		if isConversionDeploymentReady(ctx, identity) {
			klog.V(5).Info("Conversion deployment is live")
			return
		}
		klog.V(5).Info("Waiting for conversion deployment to be Ready")
		time.Sleep(conversionDeploymentRetryInterval)
	}
}

func isConversionDeploymentReady(ctx context.Context, identity string) bool {
	deploymentsClient := utils.GetKubeClient().AppsV1().Deployments(utils.SanitizeKubeResourceName(identity))
	deployment, getErr := deploymentsClient.Get(ctx, conversionWebhookName, metav1.GetOptions{})
	if getErr != nil {
		klog.V(5).Info(getErr)
		return false
	}
	return deployment.Status.ReadyReplicas >= conversionDeploymentReadinessThreshold
}
