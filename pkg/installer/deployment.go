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

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

func createControllerSecret(ctx context.Context, publicCertBytes, privateKeyBytes []byte, c *Config) error {

	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[privateKeyFileName] = privateKeyBytes
		mp[publicCertFileName] = publicCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        admissionWebhookSecretName,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Data: getCertsDataMap(),
	}

	if c.DryRun {
		return c.postProc(secret)
	}

	if _, err := client.GetKubeClient().CoreV1().Secrets(c.namespace()).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return c.postProc(secret)
}

func createControllerService(ctx context.Context, generatedSelectorValue string, c *Config) error {
	admissionWebhookPort := corev1.ServicePort{
		Port: admissionControllerWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: admissionControllerWebhookName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        validationControllerName,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{admissionWebhookPort},
			Selector: map[string]string{
				directCSISelector: generatedSelectorValue,
			},
		},
	}

	if c.DryRun {
		return c.postProc(svc)
	}

	if _, err := client.GetKubeClient().CoreV1().Services(c.namespace()).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return c.postProc(svc)
}

func createDeployment(ctx context.Context, c *Config) error {
	var replicas int32 = 3
	privileged := true
	volumes := []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", c.deploymentName()))),
		newSecretVolume(conversionCACert, conversionCACert),
		newSecretVolume(conversionKeyPair, conversionKeyPair),
	}
	directCSIVolumeMounts := []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, "/csi", corev1.MountPropagationNone, false),
		newVolumeMount(conversionCACert, conversionCADir, corev1.MountPropagationNone, false),
		newVolumeMount(conversionKeyPair, conversionCertsDir, corev1.MountPropagationNone, false),
	}

	if c.AdmissionControl {
		volumes = append(volumes, newSecretVolume(admissionControllerCertsDir, admissionWebhookSecretName))
		directCSIVolumeMounts = append(directCSIVolumeMounts, newVolumeMount(admissionControllerCertsDir, admissionCertsDir, corev1.MountPropagationNone, false))
	}

	podSpec := corev1.PodSpec{
		ServiceAccountName: c.serviceAccountName(),
		Volumes:            volumes,
		ImagePullSecrets:   c.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  csiProvisionerContainerName,
				Image: filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.getCSIProvisionerImage()),
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
					newVolumeMount(volumeNameSocketDir, "/csi", corev1.MountPropagationNone, false),
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
				Image: filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.DirectCSIContainerImage),
				Args: []string{
					fmt.Sprintf("-v=%d", logLevel),
					fmt.Sprintf("--identity=%s", c.deploymentName()),
					fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
					fmt.Sprintf("--conversion-healthz-url=%s", c.conversionHealthzURL()),
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
					{
						ContainerPort: conversionWebhookPort,
						Name:          conversionWebhookPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				ReadinessProbe: &corev1.Probe{
					Handler: getConversionHealthzHandler(),
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
				VolumeMounts: directCSIVolumeMounts,
			},
		},
	}

	if c.AdmissionControl {
		caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{admissionWehookDNSName})
		if certErr != nil {
			return certErr
		}
		c.validationWebhookCaBundle = caCertBytes

		if err := createControllerSecret(ctx, publicCertBytes, privateKeyBytes, c); err != nil {
			return err
		}
	}

	generatedSelectorValue := generateSanitizedUniqueNameFrom(c.deploymentName())

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.deploymentName(),
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.deploymentName(),
					Namespace: c.namespace(),
					Annotations: map[string]string{
						createdByLabel: directCSIPluginName,
					},
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
						webhookSelector:   selectorValueEnabled,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	deployment.ObjectMeta.Finalizers = []string{
		c.namespace() + directCSIFinalizerDeleteProtection,
	}

	if !c.DryRun {
		if _, err := client.GetKubeClient().AppsV1().Deployments(c.namespace()).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
	}

	if err := c.postProc(deployment); err != nil {
		return err
	}

	return createControllerService(ctx, generatedSelectorValue, c)
}

func installDeploymentDefault(ctx context.Context, c *Config) error {
	if err := createDeployment(ctx, c); err != nil {
		return err
	}

	if !c.DryRun {
		klog.Infof("'%s' deployment created", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallDeploymentDefault(ctx context.Context, c *Config) error {
	if err := client.GetKubeClient().CoreV1().Secrets(c.namespace()).Delete(ctx, admissionWebhookSecretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := deleteDeployment(ctx, c.namespace(), c.deploymentName()); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' controller deployment deleted", utils.Bold(c.Identity))

	return nil
}
