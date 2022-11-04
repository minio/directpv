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
	"path"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func installDeploymentDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, fmt.Sprintf("%s deployment", c.deploymentName()), createDeployment); err != nil {
		return fmt.Errorf("unable to create deployment; %v", err)
	}
	return nil
}

func uninstallDeploymentDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, fmt.Sprintf("%s deployment", c.deploymentName()), deleteDeploymentDefault); err != nil {
		return fmt.Errorf("unable to delete deployment; %v", err)
	}
	return nil
}

func deleteDeploymentDefault(ctx context.Context, c *Config) error {
	if err := deleteDeployment(ctx, c.namespace(), c.deploymentName()); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createDeployment(ctx context.Context, c *Config) error {
	var replicas int32 = 3
	privileged := true
	volumes := []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, newPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", c.deploymentName()))),
	}
	volumeMounts := []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
	}
	podSpec := corev1.PodSpec{
		ServiceAccountName: c.serviceAccountName(),
		Volumes:            volumes,
		ImagePullSecrets:   c.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  csiProvisionerContainerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.getCSIProvisionerImage()),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--timeout=300s",
					fmt.Sprintf("--csi-address=$(%s)", csiEndpointEnvVarName),
					"--leader-election",
					"--feature-gates=Topology=true",
					"--strict-topology",
				},
				Env: []corev1.EnvVar{csiEndpointEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
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
				Name:  consts.ControllerServerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.ContainerImage),
				Args: []string{
					consts.ControllerServerName,
					fmt.Sprintf("-v=%d", logLevel),
					fmt.Sprintf("--identity=%s", c.identity()),
					fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
					fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
					fmt.Sprintf("--readiness-port=%d", consts.ReadinessPort),
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports:          commonContainerPorts,
				ReadinessProbe: &corev1.Probe{ProbeHandler: readinessHandler},
				Env:            []corev1.EnvVar{kubeNodeNameEnvVar, csiEndpointEnvVar},
				VolumeMounts:   volumeMounts,
			},
		},
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
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.deploymentName(),
					Namespace: c.namespace(),
					Annotations: map[string]string{
						createdByLabel: pluginName,
					},
					Labels: map[string]string{
						selectorKey: generatedSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	deployment.Finalizers = []string{
		c.namespace() + deleteProtectionFinalizer,
	}

	if !c.DryRun {
		if _, err := k8s.KubeClient().AppsV1().Deployments(c.namespace()).Create(ctx, deployment, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return c.postProc(deployment)
}
