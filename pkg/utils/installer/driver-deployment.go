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
	"fmt"
	"path/filepath"

	"github.com/minio/direct-csi/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDeployment(
	ctx context.Context,
	identity string,
	directCSIContainerImage string,
	dryRun bool,
	registry, org string) error {

	name := sanitizeName(identity)
	ns := sanitizeName(identity)

	conversionWebhookURL := getConversionWebhookURL(identity)

	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newHostPathVolume(volumeNameSocketDir,
				newDirectCSIPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", name)),
			),
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

	objM := newObjMeta(identity, identity, "component", "controller")
	selector := &metav1.LabelSelector{}
	for k, v := range objM.Labels {
		selector = metav1.AddLabelToSelector(selector, k, v)
	}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objM,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objM,
				Spec:       podSpec,
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

	if _, err := utils.GetKubeClient().
		AppsV1().
		Deployments(ns).
		Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
