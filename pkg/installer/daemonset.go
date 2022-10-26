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
	"k8s.io/apimachinery/pkg/util/intstr"
)

func installDaemonsetDefault(ctx context.Context, c *Config) error {
	daemonSetsClient := k8s.KubeClient().AppsV1().DaemonSets(c.namespace())
	if _, err := daemonSetsClient.Get(ctx, c.daemonsetName(), metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		// Deployment already created
		return nil
	}
	return createDaemonSet(ctx, c)
}

func uninstallDaemonsetDefault(ctx context.Context, c *Config) error {
	if err := k8s.KubeClient().AppsV1().DaemonSets(c.namespace()).Delete(ctx, c.daemonsetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k8s.KubeClient().CoreV1().Secrets(c.namespace()).Delete(ctx, nodeAPIServerCertsSecretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := k8s.KubeClient().CoreV1().Secrets(c.namespace()).Delete(ctx, nodeAPIServerCASecretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createDaemonSet(ctx context.Context, c *Config) error {
	// Create cert secrets for the node api-server
	if err := generateCertSecretsForNodeAPIServer(ctx, c); err != nil {
		return err
	}
	// Create deamonset
	privileged := true
	securityContext := &corev1.SecurityContext{Privileged: &privileged}

	seccompProfileName := c.SeccompProfile
	if seccompProfileName != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &seccompProfileName,
		}
	}

	volumes := []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, newPluginsSocketDir(kubeletDirPath, c.identity())),
		newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
		newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
		newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
		newHostPathVolume(volumeNameAppRootDir, appRootDir),
		newHostPathVolume(volumeNameSysDir, volumePathSysDir),
		newHostPathVolume(volumeNameDevDir, volumePathDevDir),
		newHostPathVolume(volumeNameRunUdevData, volumePathRunUdevData),
		// node api server
		newSecretVolume(nodeAPIServerCertsDir, nodeAPIServerCertsSecretName),
	}
	volumeMounts := []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
		newVolumeMount(volumeNameMountpointDir, kubeletDirPath+"/pods", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNamePluginDir, kubeletDirPath+"/plugins", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameAppRootDir, appRootDir, corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameSysDir, volumePathSysDir, corev1.MountPropagationBidirectional, true),
		newVolumeMount(volumeNameDevDir, volumePathDevDir, corev1.MountPropagationHostToContainer, true),
		newVolumeMount(volumeNameRunUdevData, volumePathRunUdevData, corev1.MountPropagationBidirectional, true),
	}

	podSpec := corev1.PodSpec{
		ServiceAccountName: c.serviceAccountName(),
		HostIPC:            false,
		HostPID:            true,
		Volumes:            volumes,
		ImagePullSecrets:   c.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  nodeDriverRegistrarContainerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.getNodeDriverRegistrarImage()),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--csi-address=" + UnixCSIEndpoint,
					fmt.Sprintf("--kubelet-registration-path=%s",
						newPluginsSocketDir(kubeletDirPath, c.identity())+socketFile),
				},
				Env: []corev1.EnvVar{kubeNodeNameEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
					newVolumeMount(volumeNameRegistrationDir, "/registration", corev1.MountPropagationNone, false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-registrar-termination-log",
			},
			{
				Name:  consts.NodeServerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.ContainerImage),
				Args: func() []string {
					args := []string{
						consts.NodeServerName,
						fmt.Sprintf("-v=%d", logLevel),
						fmt.Sprintf("--identity=%s", c.identity()),
						fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
						fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
						fmt.Sprintf("--readiness-port=%d", consts.ReadinessPort),
						fmt.Sprintf("--metrics-port=%d", consts.MetricsPort),
					}
					return args
				}(),
				SecurityContext:          securityContext,
				Env:                      []corev1.EnvVar{kubeNodeNameEnvVar, csiEndpointEnvVar},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-termination-log",
				VolumeMounts:             volumeMounts,
				Ports: append(commonContainerPorts, corev1.ContainerPort{
					ContainerPort: consts.MetricsPort,
					Name:          metricsPortName,
					Protocol:      corev1.ProtocolTCP,
				}),
				ReadinessProbe: &corev1.Probe{ProbeHandler: readinessHandler},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 300,
					TimeoutSeconds:      5,
					PeriodSeconds:       5,
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: healthZContainerPortPath,
							Port: intstr.FromString(healthZContainerPortName),
						},
					},
				},
			},
			{
				Name:  consts.NodeAPIServerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.ContainerImage),
				Args: func() []string {
					args := []string{
						consts.NodeAPIServerName,
						fmt.Sprintf("-v=%d", logLevel),
						fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
						fmt.Sprintf("--port=%d", consts.NodeAPIPort),
					}
					return args
				}(),
				SecurityContext:          securityContext,
				Env:                      []corev1.EnvVar{kubeNodeNameEnvVar},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-termination-log",
				VolumeMounts:             append(volumeMounts, newVolumeMount(nodeAPIServerCertsDir, consts.NodeAPIServerCertsPath, corev1.MountPropagationNone, false)),
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: consts.NodeAPIPort,
						Name:          consts.NodeAPIPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				/*ReadinessProbe: &corev1.Probe{ProbeHandler: readinessHandler},
				LivenessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 300,
					TimeoutSeconds:      5,
					PeriodSeconds:       5,
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: healthZContainerPortPath,
							Port: intstr.FromString(healthZContainerPortName),
						},
					},
				},*/
			},
			{
				Name:  livenessProbeContainerName,
				Image: path.Join(c.ContainerRegistry, c.ContainerOrg, c.getLivenessProbeImage()),
				Args: []string{
					"--csi-address=" + socketDir + socketFile,
					"--health-port=" + fmt.Sprintf("%v", healthZContainerPort),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
				},
			},
		},
		NodeSelector: c.NodeSelector,
		Tolerations:  c.Tolerations,
	}

	annotations := map[string]string{
		createdByLabel: pluginName,
	}
	if c.ApparmorProfile != "" {
		annotations["container.apparmor.security.beta.kubernetes.io/"+consts.AppName] = c.ApparmorProfile
	}

	generatedSelectorValue := generateSanitizedUniqueNameFrom(c.identity())
	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.daemonsetName(),
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        c.daemonsetName(),
					Namespace:   c.namespace(),
					Annotations: annotations,
					Labels: map[string]string{
						selectorKey:     generatedSelectorValue,
						serviceSelector: selectorValueEnabled,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}

	if c.DryRun {
		return c.postProc(daemonset)
	}

	if _, err := k8s.KubeClient().AppsV1().DaemonSets(c.namespace()).Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return c.postProc(daemonset)
}

func generateCertSecretsForNodeAPIServer(ctx context.Context, c *Config) error {
	caCertBytes, publicCertBytes, privateKeyBytes, certErr := getCerts([]string{
		localHostDNS,
		// FIXME: Add clusterIP svc domain name here
	})
	if certErr != nil {
		return certErr
	}
	return createOrUpdateNodeAPIServerSecrets(ctx, caCertBytes, publicCertBytes, privateKeyBytes, c)
}

func createOrUpdateNodeAPIServerSecrets(ctx context.Context, caCertBytes, publicCertBytes, privateKeyBytes []byte, c *Config) error {
	if err := createOrUpdateSecret(ctx, nodeAPIServerCertsSecretName, map[string][]byte{
		consts.PrivateKeyFileName: privateKeyBytes,
		consts.PublicCertFileName: publicCertBytes,
	}, c); err != nil {
		return err
	}
	return createOrUpdateSecret(ctx, nodeAPIServerCASecretName, map[string][]byte{
		caCertFileName: caCertBytes,
	}, c)
}
