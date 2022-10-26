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

func installAPIServerDeploymentDefault(ctx context.Context, c *Config) error {
	if _, err := k8s.KubeClient().AppsV1().Deployments(c.namespace()).Get(ctx, c.apiServerDeploymentName(), metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		return createAPIServerDeployment(ctx, c)
	}

	return nil
}

func uninstallAPIServerDeploymentDefault(ctx context.Context, c *Config) error {
	if err := deleteDeployment(ctx, c.namespace(), c.apiServerDeploymentName()); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k8s.KubeClient().CoreV1().Secrets(c.namespace()).Delete(ctx, apiServerCertsSecretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k8s.KubeClient().CoreV1().Secrets(c.namespace()).Delete(ctx, apiServerCASecretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createAPIServerDeployment(ctx context.Context, c *Config) error {
	// Create cert secrets for the api-server
	if err := generateCertSecretsForAPIServer(ctx, c); err != nil {
		return err
	}
	// Create api-server deployment
	var replicas int32 = 1
	privileged := false
	podSpec := corev1.PodSpec{
		ServiceAccountName: c.serviceAccountName(),
		Volumes: []corev1.Volume{
			newSecretVolume(apiServerCertsDir, apiServerCertsSecretName),
			newSecretVolume(nodeAPIServerCADir, nodeAPIServerCASecretName),
		},
		ImagePullSecrets: c.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  consts.AdminServerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.ContainerImage),
				Args: []string{
					consts.AdminServerName,
					fmt.Sprintf("-v=%d", logLevel),
					fmt.Sprintf("--identity=%s", c.identity()),
					fmt.Sprintf("--port=%d", consts.APIPort),
					fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
					fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{kubeNodeNameEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(apiServerCertsDir, consts.APIServerCertsPath, corev1.MountPropagationNone, false),
					newVolumeMount(nodeAPIServerCADir, nodeAPIServerCAPath, corev1.MountPropagationNone, false),
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: consts.APIPort,
						Name:          apiPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	generatedSelectorValue := generateSanitizedUniqueNameFrom(c.apiServerDeploymentName())
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.apiServerDeploymentName(),
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.apiServerDeploymentName(),
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
		if _, err := k8s.KubeClient().AppsV1().Deployments(c.namespace()).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			return err
		}
	}

	return c.postProc(deployment)
}

func generateCertSecretsForAPIServer(ctx context.Context, c *Config) error {
	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{
		localHostDNS,
		// FIXME: Add nodeport svc domain name here
	})
	if certErr != nil {
		return certErr
	}
	return createOrUpdateAPIServerSecrets(ctx, caCertBytes, publicCertBytes, privateKeyBytes, c)
}

func createOrUpdateAPIServerSecrets(ctx context.Context, caCertBytes, publicCertBytes, privateKeyBytes []byte, c *Config) error {
	if err := createOrUpdateSecret(ctx, apiServerCertsSecretName, map[string][]byte{
		consts.PrivateKeyFileName: privateKeyBytes,
		consts.PublicCertFileName: publicCertBytes,
	}, c); err != nil {
		return err
	}
	return createOrUpdateSecret(ctx, apiServerCASecretName, map[string][]byte{
		caCertFileName: caCertBytes,
	}, c)
}
