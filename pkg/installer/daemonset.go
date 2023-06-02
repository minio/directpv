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
	"os"
	"path"
	"strings"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	"github.com/minio/directpv/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	existingDaemonset       *appsv1.DaemonSet
	existingLegacyDaemonset *appsv1.DaemonSet
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

type daemonsetTask struct{}

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

func (daemonsetTask) Execute(ctx context.Context, args *Args) error {
	return createDaemonset(ctx, args)
}

func (daemonsetTask) Delete(ctx context.Context, _ *Args) error {
	return deleteDaemonset(ctx)
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

func getVolumesAndMounts(kubeletDir, pluginSocketDir string) (volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	volumes = []corev1.Volume{
		newHostPathVolume(volumeNameSocketDir, pluginSocketDir),
		newHostPathVolume(volumeNameMountpointDir, kubeletDir+"/pods"),
		newHostPathVolume(volumeNameRegistrationDir, kubeletDir+"/plugins_registry"),
		newHostPathVolume(volumeNamePluginDir, kubeletDir+"/plugins"),
		newHostPathVolume(volumeNameAppRootDir, appRootDir),
		newHostPathVolume(volumeNameSysDir, volumePathSysDir),
		newHostPathVolume(volumeNameDevDir, volumePathDevDir),
		newHostPathVolume(volumeNameRunUdevData, volumePathRunUdevData),
		newHostPathVolume(volumeNameLegacyAppRootDir, legacyAppRootDir),
	}
	volumeMounts = []corev1.VolumeMount{
		newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
		newVolumeMount(volumeNameMountpointDir, kubeletDir+"/pods", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNamePluginDir, kubeletDir+"/plugins", corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameAppRootDir, appRootDir, corev1.MountPropagationBidirectional, false),
		newVolumeMount(volumeNameSysDir, volumePathSysDir, corev1.MountPropagationBidirectional, false),
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

func newDaemonset(podSpec corev1.PodSpec, name, appArmorProfile string, creationTimestamp metav1.Time, resourceVersion string, uid types.UID, selectorValue string) *appsv1.DaemonSet {
	annotations := map[string]string{createdByLabel: pluginName}
	if appArmorProfile != "" {
		// AppArmor profiles need to be specified per-container
		for _, container := range podSpec.Containers {
			annotations["container.apparmor.security.beta.kubernetes.io/"+container.Name] = "localhost/" + appArmorProfile
		}
	}

	if selectorValue == "" {
		selectorValue = fmt.Sprintf("%v-%v", consts.Identity, getRandSuffix())
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: creationTimestamp,
			ResourceVersion:   resourceVersion,
			UID:               uid,
			Name:              name,
			Namespace:         namespace,
			Annotations:       map[string]string{},
			Labels:            defaultLabels,
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

type daemonsetValues struct {
	update                   bool
	creationTimestamp        metav1.Time
	resourceVersion          string
	uid                      types.UID
	selectorValue            string
	nodeDriverRegistrarImage string
	containerImage           string
	livenessProbeImage       string
	imagePullSecrets         []corev1.LocalObjectReference
	nodeSelector             map[string]string
	tolerations              []corev1.Toleration
	securityContext          *corev1.SecurityContext
	appArmorProfile          string
	kubeletDir               string
}

func newDaemonsetValues(args *Args) *daemonsetValues {
	return &daemonsetValues{
		nodeDriverRegistrarImage: args.getNodeDriverRegistrarImage(),
		containerImage:           args.getContainerImage(),
		livenessProbeImage:       args.getLivenessProbeImage(),
		imagePullSecrets:         args.getImagePullSecrets(),
		nodeSelector:             args.NodeSelector,
		tolerations:              args.Tolerations,
		securityContext:          newSecurityContext(args.SeccompProfile),
		appArmorProfile:          args.AppArmorProfile,
		kubeletDir:               kubeletDirPath,
	}
}

func (values *daemonsetValues) populate(daemonset *appsv1.DaemonSet) {
	if daemonset == nil {
		return
	}

	values.update = true
	values.creationTimestamp = daemonset.CreationTimestamp
	values.resourceVersion = daemonset.ResourceVersion
	values.uid = daemonset.UID

	if daemonset.Spec.Selector != nil && daemonset.Spec.Selector.MatchLabels != nil {
		values.selectorValue = daemonset.Spec.Selector.MatchLabels[selectorKey]
	}

	if len(values.imagePullSecrets) == 0 && len(daemonset.Spec.Template.Spec.ImagePullSecrets) != 0 {
		values.imagePullSecrets = daemonset.Spec.Template.Spec.ImagePullSecrets
	}

	if len(values.nodeSelector) == 0 && len(daemonset.Spec.Template.Spec.NodeSelector) != 0 {
		values.nodeSelector = daemonset.Spec.Template.Spec.NodeSelector
	}

	if len(values.tolerations) == 0 && len(daemonset.Spec.Template.Spec.Tolerations) != 0 {
		values.tolerations = daemonset.Spec.Template.Spec.Tolerations
	}

	if values.appArmorProfile == "" && len(daemonset.Spec.Template.Annotations) != 0 {
		if value, found := daemonset.Spec.Template.Annotations["container.apparmor.security.beta.kubernetes.io/"+consts.NodeServerName]; found {
			values.appArmorProfile = strings.TrimPrefix(value, "localhost/")
		}
	}

	for _, container := range daemonset.Spec.Template.Spec.Containers {
		if container.Name != consts.NodeServerName {
			continue
		}

		if path.Dir(values.containerImage) == "quay.io/minio" {
			if dir := path.Dir(container.Image); dir != "quay.io/minio" {
				values.nodeDriverRegistrarImage = dir + "/" + path.Base(nodeDriverRegistrarImage)
				values.containerImage = dir + "/" + path.Base(values.containerImage)
				values.livenessProbeImage = dir + "/" + path.Base(livenessProbeImage)
			}
		}

		if values.securityContext.SeccompProfile == nil && container.SecurityContext.SeccompProfile != nil {
			values.securityContext.SeccompProfile = container.SecurityContext.SeccompProfile
		}

		break
	}

	_, found := os.LookupEnv("KUBELET_DIR_PATH")
	if values.kubeletDir == "/var/lib/kubelet" && !found && len(daemonset.Spec.Template.Spec.Volumes) != 0 {
		for _, volume := range daemonset.Spec.Template.Spec.Volumes {
			if volume.Name == volumeNamePluginDir && volume.HostPath != nil {
				values.kubeletDir = path.Dir(volume.HostPath.Path)
				break
			}
		}
	}
}

func doCreateDaemonset(ctx context.Context, args *Args) (err error) {
	values := newDaemonsetValues(args)
	if !args.dryRun() {
		daemonset, err := k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
			ctx, consts.NodeServerName, metav1.GetOptions{},
		)

		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if daemonset != nil && daemonset.UID != "" {
			existingDaemonset = daemonset
			if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingDaemonset)); err != nil {
				return err
			}
		}

		values.populate(existingDaemonset)
	}

	pluginSocketDir := newPluginsSocketDir(values.kubeletDir, consts.Identity)
	volumes, volumeMounts := getVolumesAndMounts(values.kubeletDir, pluginSocketDir)
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
		ImagePullSecrets:   values.imagePullSecrets,
		Containers: []corev1.Container{
			nodeDriverRegistrarContainer(values.nodeDriverRegistrarImage, pluginSocketDir),
			nodeServerContainer(values.containerImage, containerArgs, values.securityContext, volumeMounts),
			nodeControllerContainer(values.containerImage, nodeControllerArgs, values.securityContext, volumeMounts),
			livenessProbeContainer(values.livenessProbeImage),
		},
		NodeSelector: values.nodeSelector,
		Tolerations:  values.tolerations,
	}

	daemonset := newDaemonset(
		podSpec, consts.NodeServerName, values.appArmorProfile, values.creationTimestamp,
		values.resourceVersion, values.uid, values.selectorValue,
	)

	if args.dryRun() {
		args.DryRunPrinter(daemonset)
		return nil
	}

	if values.update {
		_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Update(
			ctx, daemonset, metav1.UpdateOptions{},
		)
	} else {
		_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Create(
			ctx, daemonset, metav1.CreateOptions{},
		)
	}
	if err != nil {
		return err
	}

	_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(daemonset))
	return err
}

func doCreateLegacyDaemonset(ctx context.Context, args *Args) (err error) {
	values := newDaemonsetValues(args)
	if !args.dryRun() {
		daemonset, err := k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
			ctx, consts.LegacyNodeServerName, metav1.GetOptions{},
		)

		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if daemonset != nil && daemonset.UID != "" {
			existingLegacyDaemonset = daemonset
			if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingLegacyDaemonset)); err != nil {
				return err
			}
		}

		values.populate(existingLegacyDaemonset)
	}

	pluginSocketDir := newPluginsSocketDir(values.kubeletDir, legacyclient.Identity)
	volumes, volumeMounts := getVolumesAndMounts(values.kubeletDir, pluginSocketDir)
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
		ImagePullSecrets:   values.imagePullSecrets,
		Containers: []corev1.Container{
			nodeDriverRegistrarContainer(values.nodeDriverRegistrarImage, pluginSocketDir),
			nodeServerContainer(values.containerImage, containerArgs, values.securityContext, volumeMounts),
			livenessProbeContainer(values.livenessProbeImage),
		},
		NodeSelector: values.nodeSelector,
		Tolerations:  values.tolerations,
	}

	daemonset := newDaemonset(
		podSpec, consts.LegacyNodeServerName, values.appArmorProfile, values.creationTimestamp,
		values.resourceVersion, values.uid, values.selectorValue,
	)

	if args.dryRun() {
		args.DryRunPrinter(daemonset)
		return nil
	}

	if values.update {
		_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Update(
			ctx, daemonset, metav1.UpdateOptions{},
		)
	} else {
		_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Create(
			ctx, daemonset, metav1.CreateOptions{},
		)
	}
	if err != nil {
		return err
	}

	_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(daemonset))
	return err
}

func createDaemonset(ctx context.Context, args *Args) (err error) {
	if args.dryRun() {
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

	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s Daemonset", consts.NodeServerName), 1, nil) {
		return errSendProgress
	}
	if err := doCreateDaemonset(ctx, args); err != nil {
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
	if err := doCreateLegacyDaemonset(ctx, args); err != nil {
		return err
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s Daemonset", consts.LegacyNodeServerName), 2, daemonsetComponent(consts.LegacyNodeServerName)) {
		return errSendProgress
	}
	return nil
}

func doRevertDaemonSet(ctx context.Context) error {
	if existingDaemonset == nil {
		return nil
	}

	daemonset, err := k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
		ctx, consts.NodeServerName, metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	existingDaemonset.ResourceVersion = daemonset.ResourceVersion
	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Update(
		ctx, existingDaemonset, metav1.UpdateOptions{},
	)
	return err
}

func doRevertLegacyDaemonSet(ctx context.Context) error {
	if existingLegacyDaemonset == nil {
		return nil
	}

	daemonset, err := k8s.KubeClient().AppsV1().DaemonSets(namespace).Get(
		ctx, consts.LegacyNodeServerName, metav1.GetOptions{},
	)
	if err != nil {
		return err
	}

	existingLegacyDaemonset.ResourceVersion = daemonset.ResourceVersion
	_, err = k8s.KubeClient().AppsV1().DaemonSets(namespace).Update(
		ctx, existingLegacyDaemonset, metav1.UpdateOptions{},
	)
	return err
}

func revertDaemonSet(ctx context.Context, args *Args) error {
	var err, legacyErr error
	err = doRevertDaemonSet(ctx)
	if args.Legacy {
		legacyErr = doRevertLegacyDaemonSet(ctx)
	}

	if err != nil && legacyErr != nil {
		return fmt.Errorf("unable to revert; DaemonSet: %v; legacy DaemonSet: %v", err, legacyErr)
	}

	if err != nil {
		return err
	}

	return legacyErr
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
