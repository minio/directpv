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

	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateDaemonSet(
	ctx context.Context,
	identity string,
	directCSIContainerImage string,
	dryRun bool,
	registry, org string,
	loopBackOnly bool) error {

	name := sanitizeName(identity)
	ns := sanitizeName(identity)
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
	}

	objM := newObjMeta(identity, identity, "component", "driver")
	selector := &metav1.LabelSelector{}
	for k, v := range objM.Labels {
		selector = metav1.AddLabelToSelector(selector, k, v)
	}
	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objM,
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objM,
				Spec:       podSpec,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}

	if dryRun {
		return utils.LogYAML(daemonset)
	}

	if _, err := utils.GetKubeClient().
		AppsV1().
		DaemonSets(ns).
		Create(ctx, daemonset, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
