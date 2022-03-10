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

func installDaemonsetDefault(ctx context.Context, c *Config) error {
	if err := createDaemonSet(ctx, c); err != nil {
		return err
	}

	if !c.DryRun {
		klog.Infof("'%s' daemonset created", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallDaemonsetDefault(ctx context.Context, c *Config) error {
	if err := client.GetKubeClient().AppsV1().DaemonSets(c.namespace()).Delete(ctx, c.daemonsetName(), metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

	}
	klog.Infof("'%s' daemonset deleted", utils.Bold(c.Identity))
	return nil
}

func createDaemonSet(ctx context.Context, c *Config) error {

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
		newHostPathVolume(volumeNameSocketDir, newDirectCSIPluginsSocketDir(kubeletDirPath, c.daemonsetName())),
		newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
		newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
		newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
		newHostPathVolume(volumeNameCSIRootDir, csiRootPath),
		newSecretVolume(conversionCACert, conversionCACert),
		newSecretVolume(conversionKeyPair, conversionKeyPair),
	}
	volumeMounts := []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, "/csi", corev1.MountPropagationNone, false),
		newVolumeMount(volumeNameMountpointDir, kubeletDirPath+"/pods", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNamePluginDir, kubeletDirPath+"/plugins", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameCSIRootDir, csiRootPath, corev1.MountPropagationBidirectional, false),
		newVolumeMount(conversionCACert, conversionCADir, corev1.MountPropagationNone, false),
		newVolumeMount(conversionKeyPair, conversionCertsDir, corev1.MountPropagationNone, false),
	}

	volumes = append(volumes, newHostPathVolume(volumeNameSysDir, volumePathSysDir))
	volumeMounts = append(volumeMounts, newVolumeMount(volumeNameSysDir, volumePathSysDir, corev1.MountPropagationBidirectional, true))

	volumes = append(volumes, newHostPathVolume(volumeNameDevDir, volumePathDevDir))
	volumeMounts = append(volumeMounts, newVolumeMount(volumeNameDevDir, volumePathDevDir, corev1.MountPropagationHostToContainer, true))

	if c.DynamicDriveDiscovery {
		volumes = append(volumes, newHostPathVolume(volumeNameRunUdevData, volumePathRunUdevData))
		volumeMounts = append(volumeMounts, newVolumeMount(volumeNameRunUdevData, volumePathRunUdevData, corev1.MountPropagationBidirectional, true))
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
				Image: filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.getNodeDriverRegistrarImage()),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--csi-address=unix:///csi/csi.sock",
					fmt.Sprintf("--kubelet-registration-path=%s",
						newDirectCSIPluginsSocketDir(kubeletDirPath, c.daemonsetName())+"/csi.sock"),
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
					newVolumeMount(volumeNameSocketDir, "/csi", corev1.MountPropagationNone, false),
					newVolumeMount(volumeNameRegistrationDir, "/registration", corev1.MountPropagationNone, false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-registrar-termination-log",
			},
			{
				Name:  directCSIContainerName,
				Image: filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.DirectCSIContainerImage),
				Args: func() []string {
					args := []string{
						fmt.Sprintf("--identity=%s", c.daemonsetName()),
						fmt.Sprintf("-v=%d", logLevel),
						fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
						fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
						fmt.Sprintf("--conversion-healthz-url=%s", c.conversionHealthzURL()),
						fmt.Sprintf("--metrics-port=%d", metricsPort),
						"--driver",
					}
					if c.LoopbackMode {
						args = append(args, "--loopback-only")
					}
					if c.DynamicDriveDiscovery {
						args = append(args, "--dynamic-drive-discovery")
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
				VolumeMounts:             volumeMounts,
				Ports: []corev1.ContainerPort{
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
					{
						ContainerPort: metricsPort,
						Name:          metricsPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				ReadinessProbe: &corev1.Probe{
					Handler: getConversionHealthzHandler(),
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
				Image: filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.getLivenessProbeImage()),
				Args: []string{
					"--csi-address=/csi/csi.sock",
					"--health-port=9898",
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, "/csi", corev1.MountPropagationNone, false),
				},
			},
		},
		NodeSelector: c.NodeSelector,
		Tolerations:  c.Tolerations,
	}

	if c.DynamicDriveDiscovery {
		podSpec.HostNetwork = true
		podSpec.DNSPolicy = corev1.DNSClusterFirstWithHostNet
		args := []string{
			fmt.Sprintf("--identity=%s", c.daemonsetName()),
			fmt.Sprintf("-v=%d", logLevel),
			fmt.Sprintf("--endpoint=$(%s)", endpointEnvVarCSI),
			fmt.Sprintf("--node-id=$(%s)", kubeNodeNameEnvVar),
			"--dynamic-drive-handler",
		}
		if c.LoopbackMode {
			args = append(args, "--loopback-only")
		}
		podSpec.Containers = append(podSpec.Containers, corev1.Container{
			Name:            directPVDriveDiscoveryContainerName,
			Image:           filepath.Join(c.DirectCSIContainerRegistry, c.DirectCSIContainerOrg, c.DirectCSIContainerImage),
			Args:            args,
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
			},
			TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
			TerminationMessagePath:   "/var/log/driver-termination-log",
			VolumeMounts: []corev1.VolumeMount{
				newVolumeMount(volumeNameSysDir, volumePathSysDir, corev1.MountPropagationBidirectional, true),
				newVolumeMount(volumeNameDevDir, volumePathDevDir, corev1.MountPropagationHostToContainer, true),
				newVolumeMount(volumeNameRunUdevData, volumePathRunUdevData, corev1.MountPropagationBidirectional, true),
			},
		})
	}

	annotations := map[string]string{
		createdByLabel: directCSIPluginName,
	}
	if c.ApparmorProfile != "" {
		annotations["container.apparmor.security.beta.kubernetes.io/direct-csi"] = c.ApparmorProfile
	}

	generatedSelectorValue := generateSanitizedUniqueNameFrom(c.daemonsetName())
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
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, directCSISelector, generatedSelectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        c.daemonsetName(),
					Namespace:   c.namespace(),
					Annotations: annotations,
					Labels: map[string]string{
						directCSISelector: generatedSelectorValue,
						webhookSelector:   selectorValueEnabled,
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

	if _, err := client.GetKubeClient().AppsV1().DaemonSets(c.namespace()).Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return c.postProc(daemonset)
}
