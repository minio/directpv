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
	"io"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	nodeAPIServerCertsSecretName = "nodeapiservercerts"
	nodeAPIServerCASecretName    = "nodeapiservercacert"
	volumeNameMountpointDir      = "mountpoint-dir"
	volumeNameRegistrationDir    = "registration-dir"
	volumeNamePluginDir          = "plugins-dir"
	volumeNameAppRootDir         = consts.AppName + "-common-root"
	appRootDir                   = consts.AppRootDir + "/"
	volumeNameSysDir             = "sysfs"
	volumeNameDevDir             = "devfs"
	volumePathDevDir             = "/dev"
	volumeNameRunUdevData        = "run-udev-data-dir"
	volumePathRunUdevData        = consts.UdevDataDir
	socketFile                   = "/csi.sock"
	nodeAPIServerCertsDir        = "node-api-server-certs"
)

func createOrUpdateNodeAPIServerSecrets(ctx context.Context, args *Args) error {
	caCertBytes, publicCertBytes, privateKeyBytes, err := getCerts(
		localhost,
		// FIXME: Add clusterIP service domain name here
	)
	if err != nil {
		return err
	}

	err = createOrUpdateSecret(
		ctx,
		args,
		nodeAPIServerCertsSecretName,
		map[string][]byte{
			consts.PrivateKeyFileName: privateKeyBytes,
			consts.PublicCertFileName: publicCertBytes,
		},
	)
	if err != nil {
		return err
	}

	return createOrUpdateSecret(
		ctx,
		args,
		nodeAPIServerCASecretName,
		map[string][]byte{
			caCertFileName: caCertBytes,
		},
	)
}

func doCreateDaemonset(ctx context.Context, args *Args) error {
	if err := createOrUpdateNodeAPIServerSecrets(ctx, args); err != nil {
		return err
	}

	privileged := true
	securityContext := &corev1.SecurityContext{Privileged: &privileged}

	if args.SeccompProfile != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &args.SeccompProfile,
		}
	}

	volumes := []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, newPluginsSocketDir(kubeletDirPath, consts.Identity)),
		newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
		newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
		newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
		newHostPathVolume(volumeNameAppRootDir, appRootDir),
		newHostPathVolume(volumeNameSysDir, volumePathSysDir),
		newHostPathVolume(volumeNameDevDir, volumePathDevDir),
		newHostPathVolume(volumeNameRunUdevData, volumePathRunUdevData),
		newSecretVolume(nodeAPIServerCertsDir, nodeAPIServerCertsSecretName), // node api server
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
		ServiceAccountName: consts.Identity,
		HostIPC:            false,
		HostPID:            true,
		Volumes:            volumes,
		ImagePullSecrets:   args.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  "node-driver-registrar",
				Image: args.getNodeDriverRegistrarImage(),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					fmt.Sprintf("--csi-address=%v", UnixCSIEndpoint),
					fmt.Sprintf(
						"--kubelet-registration-path=%s%s",
						newPluginsSocketDir(kubeletDirPath, consts.Identity), socketFile,
					),
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
				Image: args.getContainerImage(),
				Args: func() []string {
					args := []string{
						consts.NodeServerName,
						fmt.Sprintf("-v=%d", logLevel),
						fmt.Sprintf("--identity=%s", consts.Identity),
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
					Name:          "metrics",
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
				Image: args.getContainerImage(),
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
				Name:  "liveness-probe",
				Image: args.getLivenessProbeImage(),
				Args: []string{
					fmt.Sprintf("--csi-address=%v%v", socketDir, socketFile),
					fmt.Sprintf("--health-port=%v", healthZContainerPort),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
				},
			},
		},
		NodeSelector: args.NodeSelector,
		Tolerations:  args.Tolerations,
	}

	annotations := map[string]string{
		createdByLabel: pluginName,
	}
	if args.AppArmorProfile != "" {
		annotations["container.apparmor.security.beta.kubernetes.io/"+consts.AppName] = args.AppArmorProfile
	}

	selectorValue := fmt.Sprintf("%v-%v", consts.Identity, getRandSuffix())
	daemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.NodeServerName,
			Namespace:   consts.Identity,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, selectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        consts.NodeServerName,
					Namespace:   consts.Identity,
					Annotations: annotations,
					Labels: map[string]string{
						selectorKey:     selectorValue,
						serviceSelector: selectorValueEnabled,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(daemonset))
		return nil
	}

	_, err := k8s.KubeClient().AppsV1().DaemonSets(consts.Identity).Create(
		ctx, daemonset, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(daemonset))
	return err
}

func createDaemonset(ctx context.Context, args *Args) error {
	if args.DryRun {
		return doCreateDaemonset(ctx, args)
	}

	_, err := k8s.KubeClient().AppsV1().DaemonSets(consts.Identity).Get(
		ctx, consts.NodeServerName, metav1.GetOptions{},
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return doCreateDaemonset(ctx, args)
	}

	return nil
}

func deleteDaemonset(ctx context.Context) error {
	err := k8s.KubeClient().AppsV1().DaemonSets(consts.Identity).Delete(
		ctx, consts.NodeServerName, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().CoreV1().Secrets(consts.Identity).Delete(ctx, nodeAPIServerCertsSecretName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().CoreV1().Secrets(consts.Identity).Delete(ctx, nodeAPIServerCASecretName, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
