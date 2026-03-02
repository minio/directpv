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

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/client"
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
	kubeletPodsDirVolumeName    = "mountpoint-dir"
	registrationDirVolumeName   = "registration-dir"
	kubeletPluginsDirVolumeName = "plugins-dir"
	socketFile                  = "/csi.sock"
	totalDaemonsetSteps         = 2
)

type daemonsetTask struct {
	client *client.Client
}

func (daemonsetTask) Name() string {
	return "Daemonset"
}

func (daemonsetTask) Start(ctx context.Context, args *Args) error {
	steps := 1
	if args.Legacy {
		steps++
	}
	if !sendStartMessage(ctx, args.ProgressCh, steps) {
		return errSendProgress
	}
	return nil
}

func (daemonsetTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (t daemonsetTask) Execute(ctx context.Context, args *Args) error {
	return t.createDaemonset(ctx, args)
}

func (t daemonsetTask) Delete(ctx context.Context, _ *Args) error {
	return t.deleteDaemonset(ctx)
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
		k8s.NewHostPathVolume(csiDirVolumeName, pluginSocketDir),
		k8s.NewHostPathVolume(kubeletPodsDirVolumeName, kubeletDirPath+"/pods"),
		k8s.NewHostPathVolume(registrationDirVolumeName, kubeletDirPath+"/plugins_registry"),
		k8s.NewHostPathVolume(kubeletPluginsDirVolumeName, kubeletDirPath+"/plugins"),
		k8s.NewHostPathVolume(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath),
		k8s.NewHostPathVolume(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath),
		k8s.NewHostPathVolume(consts.SysDirVolumeName, consts.SysDirVolumePath),
		k8s.NewHostPathVolume(consts.DevDirVolumeName, consts.DevDirVolumePath),
		k8s.NewHostPathVolume(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath),
	}
	volumeMounts = []corev1.VolumeMount{
		k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
		k8s.NewVolumeMount(kubeletPodsDirVolumeName, kubeletDirPath+"/pods", corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(kubeletPluginsDirVolumeName, kubeletDirPath+"/plugins", corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.AppRootDirVolumeName, consts.AppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.LegacyAppRootDirVolumeName, consts.LegacyAppRootDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.SysDirVolumeName, consts.SysDirVolumePath, corev1.MountPropagationBidirectional, false),
		k8s.NewVolumeMount(consts.DevDirVolumeName, consts.DevDirVolumePath, corev1.MountPropagationHostToContainer, true),
		k8s.NewVolumeMount(consts.RunUdevDataVolumeName, consts.RunUdevDataVolumePath, corev1.MountPropagationBidirectional, true),
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
			k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
			k8s.NewVolumeMount(registrationDirVolumeName, "/registration", corev1.MountPropagationNone, false),
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
		ReadinessProbe: &corev1.Probe{
			FailureThreshold:    5,
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			ProbeHandler:        readinessHandler,
		},
		LivenessProbe: &corev1.Probe{
			FailureThreshold:    5,
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
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
			fmt.Sprintf("--csi-address=%v%v", csiDirVolumePath, socketFile),
			fmt.Sprintf("--health-port=%v", healthZContainerPort),
		},
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		TerminationMessagePath:   "/var/log/driver-liveness-termination-log",
		VolumeMounts: []corev1.VolumeMount{
			k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
		},
	}
}

func newDaemonset(podSpec corev1.PodSpec, name, selectorValue string, args *Args) *appsv1.DaemonSet {
	annotations := map[string]string{createdByLabel: pluginName}
	if args.AppArmorProfile != "" {
		// AppArmor profiles need to be specified per-container
		for _, container := range podSpec.Containers {
			annotations["container.apparmor.security.beta.kubernetes.io/"+container.Name] = "localhost/" + args.AppArmorProfile
		}
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				string(directpvtypes.ImageTagLabelKey): args.imageTag,
			},
			Labels: defaultLabels,
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
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
		},
		Status: appsv1.DaemonSetStatus{},
	}
}

func (t daemonsetTask) doCreateDaemonset(ctx context.Context, args *Args) (err error) {
	securityContext := newSecurityContext(args.SeccompProfile)
	pluginSocketDir := newPluginsSocketDir(kubeletDirPath, consts.Identity)
	volumes, volumeMounts := getVolumesAndMounts(pluginSocketDir)
	containerArgs := []string{
		consts.NodeServerName,
		fmt.Sprintf("-v=%d", logLevel),
		"--identity=" + consts.Identity,
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

	var selectorValue string
	if !args.DryRun {
		daemonset, err := t.client.Kube().AppsV1().DaemonSets(namespace).Get(
			ctx, consts.NodeServerName, metav1.GetOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			if !args.Declarative {
				return nil
			}

			if daemonset.Spec.Selector != nil && daemonset.Spec.Selector.MatchLabels != nil {
				selectorValue = daemonset.Spec.Selector.MatchLabels[selectorKey]
			}
		}
	}
	if selectorValue == "" {
		selectorValue = fmt.Sprintf("%v-%v", consts.Identity, consts.NodeServerName)
	}

	daemonset := newDaemonset(podSpec, consts.NodeServerName, selectorValue, args)

	if !args.DryRun && !args.Declarative {
		_, err = t.client.Kube().AppsV1().DaemonSets(namespace).Create(
			ctx, daemonset, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(daemonset)
}

func (t daemonsetTask) doCreateLegacyDaemonset(ctx context.Context, args *Args) (err error) {
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

	var selectorValue string
	if !args.DryRun {
		daemonset, err := t.client.Kube().AppsV1().DaemonSets(namespace).Get(
			ctx, consts.LegacyNodeServerName, metav1.GetOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			if !args.Declarative {
				return nil
			}

			if daemonset.Spec.Selector != nil && daemonset.Spec.Selector.MatchLabels != nil {
				selectorValue = daemonset.Spec.Selector.MatchLabels[selectorKey]
			}
		}
	}
	if selectorValue == "" {
		selectorValue = fmt.Sprintf("%v-%v", consts.Identity, consts.LegacyNodeServerName)
	}

	daemonset := newDaemonset(podSpec, consts.LegacyNodeServerName, selectorValue, args)

	if !args.DryRun && !args.Declarative {
		_, err = t.client.Kube().AppsV1().DaemonSets(namespace).Create(
			ctx, daemonset, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(daemonset)
}

func (t daemonsetTask) createDaemonset(ctx context.Context, args *Args) (err error) {
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s Daemonset", consts.NodeServerName), 1, nil) {
		return errSendProgress
	}
	if err := t.doCreateDaemonset(ctx, args); err != nil {
		return err
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s Daemonset", consts.NodeServerName), 1, daemonsetComponent(consts.NodeServerName)) {
		return errSendProgress
	}

	if !args.Legacy {
		return nil
	}

	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s Daemonset", consts.LegacyNodeServerName), 2, nil) {
		return errSendProgress
	}
	if err := t.doCreateLegacyDaemonset(ctx, args); err != nil {
		return err
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s Daemonset", consts.LegacyNodeServerName), 2, daemonsetComponent(consts.LegacyNodeServerName)) {
		return errSendProgress
	}

	return nil
}

func (t daemonsetTask) deleteDaemonset(ctx context.Context) error {
	err := t.client.Kube().AppsV1().DaemonSets(namespace).Delete(
		ctx, consts.NodeServerName, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	err = t.client.Kube().AppsV1().DaemonSets(namespace).Delete(
		ctx, consts.LegacyNodeServerName, metav1.DeleteOptions{},
	)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}
