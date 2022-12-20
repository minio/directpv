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
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	volumeNameMountpointDir    = "mountpoint-dir"
	volumeNameRegistrationDir  = "registration-dir"
	volumeNamePluginDir        = "plugins-dir"
	volumeNameAppRootDir       = consts.AppName + "-common-root"
	volumeNameLegacyAppRootDir = "direct-csi-common-root"
	appRootDir                 = consts.AppRootDir + "/"
	legacyAppRootDir           = "/var/lib/direct-csi/"
	volumeNameSysDir           = "sysfs"
	volumeNameDevDir           = "devfs"
	volumePathDevDir           = "/dev"
	volumeNameRunUdevData      = "run-udev-data-dir"
	volumePathRunUdevData      = consts.UdevDataDir
	socketFile                 = "/csi.sock"
	totalDaemonsetSteps        = 2
)

var daemonsetStepsCompleted int

func daemonsetTask(done bool) *Task {
	if !done {
		daemonsetStepsCompleted++
	}
	return newTask(totalDaemonsetSteps, daemonsetStepsCompleted, done)
}

func newSecurityContext(seccompProfile string) *corev1.SecurityContext {
	privileged := true
	securityContext := &corev1.SecurityContext{Privileged: &privileged}
	if seccompProfile != "" {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &seccompProfile,
		}
	}

	return securityContext
}

func getVolumesAndMounts(pluginSocketDir string) (volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	volumes = []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, pluginSocketDir),
		newHostPathVolume(volumeNameMountpointDir, kubeletDirPath+"/pods"),
		newHostPathVolume(volumeNameRegistrationDir, kubeletDirPath+"/plugins_registry"),
		newHostPathVolume(volumeNamePluginDir, kubeletDirPath+"/plugins"),
		newHostPathVolume(volumeNameAppRootDir, appRootDir),
		newHostPathVolume(volumeNameSysDir, volumePathSysDir),
		newHostPathVolume(volumeNameDevDir, volumePathDevDir),
		newHostPathVolume(volumeNameRunUdevData, volumePathRunUdevData),
		newHostPathVolume(volumeNameLegacyAppRootDir, legacyAppRootDir),
	}
	volumeMounts = []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
		newVolumeMount(volumeNameMountpointDir, kubeletDirPath+"/pods", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNamePluginDir, kubeletDirPath+"/plugins", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameAppRootDir, appRootDir, corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameSysDir, volumePathSysDir, corev1.MountPropagationBidirectional, true),
		newVolumeMount(volumeNameDevDir, volumePathDevDir, corev1.MountPropagationHostToContainer, true),
		newVolumeMount(volumeNameRunUdevData, volumePathRunUdevData, corev1.MountPropagationBidirectional, true),
		newVolumeMount(volumeNameLegacyAppRootDir, legacyAppRootDir, corev1.MountPropagationBidirectional, false),
	}

	return
}

func nodeDriverRegistrarContainer(image, pluginSocketDir string) corev1.Container {
	return corev1.Container{
		Name:  "node-driver-registrar",
		Image: image,
		Args: []string{
			fmt.Sprintf("--v=%d", logLevel),
			fmt.Sprintf("--csi-address=%v", UnixCSIEndpoint),
			fmt.Sprintf("--kubelet-registration-path=%s%s", pluginSocketDir, socketFile),
		},
		Env: []corev1.EnvVar{kubeNodeNameEnvVar},
		VolumeMounts: []corev1.VolumeMount{
			newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
			newVolumeMount(volumeNameRegistrationDir, "/registration", corev1.MountPropagationNone, false),
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		TerminationMessagePath:   "/var/log/driver-registrar-termination-log",
	}
}

func nodeServerContainer(image string, args []string, securityContext *corev1.SecurityContext, volumeMounts []corev1.VolumeMount) corev1.Container {
	return corev1.Container{
		Name:                     consts.NodeServerName,
		Image:                    image,
		Args:                     args,
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
	}
}

func nodeControllerContainer(image string, args []string, securityContext *corev1.SecurityContext, volumeMounts []corev1.VolumeMount) corev1.Container {
	return corev1.Container{
		Name:                     consts.NodeControllerName,
		Image:                    image,
		Args:                     args,
		SecurityContext:          securityContext,
		Env:                      []corev1.EnvVar{kubeNodeNameEnvVar},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		TerminationMessagePath:   "/var/log/driver-termination-log",
		VolumeMounts:             volumeMounts,
	}
}

func livenessProbeContainer(image string) corev1.Container {
	return corev1.Container{
		Name:  "liveness-probe",
		Image: image,
		Args: []string{
			fmt.Sprintf("--csi-address=%v%v", socketDir, socketFile),
			fmt.Sprintf("--health-port=%v", healthZContainerPort),
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
		VolumeMounts: []corev1.VolumeMount{
			newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
		},
	}
}

func newDaemonset(podSpec corev1.PodSpec, name, appArmorProfile string) *appsv1.DaemonSet {
	annotations := map[string]string{createdByLabel: pluginName}
	if appArmorProfile != "" {
		annotations["container.apparmor.security.beta.kubernetes.io/"+consts.AppName] = appArmorProfile
	}
	selectorValue := fmt.Sprintf("%v-%v", consts.Identity, getRandSuffix())

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, selectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Namespace:   namespace,
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
}

func doCreateDaemonset(ctx context.Context, args *Args) (err error) {
	sendProgressEvent(args.Progress, fmt.Sprintf("Creating %s Daemonset", consts.NodeServerName), nil)
	defer func() {
		if err == nil {
			installedComponents = append(installedComponents, daemonsetComponent(consts.NodeServerName))
			sendProgressEvent(args.Progress, fmt.Sprintf("Created %s Daemonset", consts.NodeServerName), daemonsetTask(false))
		}
	}()
	securityContext := newSecurityContext(args.SeccompProfile)
	pluginSocketDir := newPluginsSocketDir(kubeletDirPath, consts.Identity)
	volumes, volumeMounts := getVolumesAndMounts(pluginSocketDir)
	containerArgs := []string{
		consts.NodeServerName,
		fmt.Sprintf("-v=%d", logLevel),
		fmt.Sprintf("--identity=%s", consts.Identity),
		fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
		fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
		fmt.Sprintf("--readiness-port=%d", consts.ReadinessPort),
		fmt.Sprintf("--metrics-port=%d", consts.MetricsPort),
	}
	nodeControllerArgs := []string{
		consts.NodeControllerName,
		fmt.Sprintf("-v=%d", logLevel),
		fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
	}

	podSpec := corev1.PodSpec{
		ServiceAccountName: consts.Identity,
		HostIPC:            false,
		HostPID:            true,
		Volumes:            volumes,
		ImagePullSecrets:   args.getImagePullSecrets(),
		Containers: []corev1.Container{
			nodeDriverRegistrarContainer(args.getNodeDriverRegistrarImage(), pluginSocketDir),
			nodeServerContainer(args.getContainerImage(), containerArgs, securityContext, volumeMounts),
			nodeControllerContainer(args.getContainerImage(), nodeControllerArgs, securityContext, volumeMounts),
			livenessProbeContainer(args.getLivenessProbeImage()),
		},
		NodeSelector: args.NodeSelector,
		Tolerations:  args.Tolerations,
	}

	daemonset := newDaemonset(podSpec, consts.NodeServerName, args.AppArmorProfile)

	if args.DryRun {
		fmt.Print(mustGetYAML(daemonset))
		return nil
	}

	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Create(
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

func doCreateLegacyDaemonset(ctx context.Context, args *Args) (err error) {
	sendProgressEvent(args.Progress, fmt.Sprintf("Creating %s Daemonset", consts.LegacyNodeServerName), nil)
	defer func() {
		if err == nil {
			installedComponents = append(installedComponents, daemonsetComponent(consts.LegacyNodeServerName))
			sendProgressEvent(args.Progress, fmt.Sprintf("Created %s Daemonset", consts.LegacyNodeServerName), daemonsetTask(false))
		}
	}()
	securityContext := newSecurityContext(args.SeccompProfile)
	pluginSocketDir := newPluginsSocketDir(kubeletDirPath, legacyclient.Identity)
	volumes, volumeMounts := getVolumesAndMounts(pluginSocketDir)
	containerArgs := []string{
		consts.LegacyNodeServerName,
		fmt.Sprintf("-v=%d", logLevel),
		fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
		fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
		fmt.Sprintf("--readiness-port=%d", consts.ReadinessPort),
	}

	podSpec := corev1.PodSpec{
		ServiceAccountName: consts.Identity,
		HostIPC:            false,
		HostPID:            true,
		Volumes:            volumes,
		ImagePullSecrets:   args.getImagePullSecrets(),
		Containers: []corev1.Container{
			nodeDriverRegistrarContainer(args.getNodeDriverRegistrarImage(), pluginSocketDir),
			nodeServerContainer(args.getContainerImage(), containerArgs, securityContext, volumeMounts),
			livenessProbeContainer(args.getLivenessProbeImage()),
		},
		NodeSelector: args.NodeSelector,
		Tolerations:  args.Tolerations,
	}

	daemonset := newDaemonset(podSpec, consts.LegacyNodeServerName, args.AppArmorProfile)

	if args.DryRun {
		fmt.Print(mustGetYAML(daemonset))
		return nil
	}

	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Create(
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

func createDaemonset(ctx context.Context, args *Args) (err error) {
	sendProgressEvent(args.Progress, "Creating Daemonset", nil)
	defer func() {
		if err == nil {
			sendProgressEvent(args.Progress, "Created Daemonset", daemonsetTask(true))
		}
	}()
	if args.DryRun {
		if err := doCreateDaemonset(ctx, args); err != nil {
			return err
		}

		if args.Legacy {
			if err := doCreateLegacyDaemonset(ctx, args); err != nil {
				return err
			}
		}

		return nil
	}

	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
		ctx, consts.NodeServerName, metav1.GetOptions{},
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := doCreateDaemonset(ctx, args); err != nil {
			return err
		}
	}

	if !args.Legacy {
		return nil
	}

	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
		ctx, consts.LegacyNodeServerName, metav1.GetOptions{},
	)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := doCreateLegacyDaemonset(ctx, args); err != nil {
			return err
		}
	}

	return nil
}

func deleteDaemonset(ctx context.Context) error {
	err := k8s.KubeClient().AppsV1().DaemonSets(namespace).Delete(
		ctx, consts.NodeServerName, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Delete(
		ctx, consts.LegacyNodeServerName, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
