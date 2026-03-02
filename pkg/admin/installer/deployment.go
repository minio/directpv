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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const deploymentFinalizer = consts.Identity + "/delete-protection"

type deploymentTask struct {
	client *client.Client
}

func (deploymentTask) Name() string {
	return "Deployment"
}

func (deploymentTask) Start(ctx context.Context, args *Args) error {
	steps := 1
	if args.Legacy {
		steps++
	}
	if !sendStartMessage(ctx, args.ProgressCh, steps) {
		return errSendProgress
	}
	return nil
}

func (deploymentTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (t deploymentTask) Execute(ctx context.Context, args *Args) error {
	return t.createDeployment(ctx, args)
}

func (t deploymentTask) Delete(ctx context.Context, _ *Args) error {
	return t.deleteDeployment(ctx)
}

func (t deploymentTask) doCreateDeployment(ctx context.Context, args *Args, legacy bool, step int) (err error) {
	name := consts.ControllerServerName
	containerArgs := []string{name, "--identity=" + consts.Identity}
	if legacy {
		name = consts.LegacyControllerServerName
		containerArgs = []string{name}
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s Deployment", name), step, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s Deployment", name), step, deploymentComponent(name)) {
				err = errSendProgress
			}
		}
	}()
	containerArgs = append(
		containerArgs,
		[]string{
			fmt.Sprintf("-v=%d", logLevel),
			fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
			fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
			fmt.Sprintf("--readiness-port=%d", consts.ReadinessPort),
		}...,
	)

	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: consts.Identity,
		Volumes: []corev1.Volume{
			k8s.NewHostPathVolume(
				csiDirVolumeName,
				newPluginsSocketDir(kubeletDirPath, consts.ControllerServerName+"-controller"),
			),
		},
		ImagePullSecrets: args.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  "csi-provisioner",
				Image: args.getCSIProvisionerImage(),
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
					k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
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
				Name:  "csi-resizer",
				Image: args.getCSIResizerImage(),
				Args: []string{
					fmt.Sprintf("--v=%d", logLevel),
					"--timeout=300s",
					fmt.Sprintf("--csi-address=$(%s)", csiEndpointEnvVarName),
					"--leader-election",
				},
				Env: []corev1.EnvVar{csiEndpointEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
				},
				TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
				TerminationMessagePath:   "/var/log/controller-csi-resizer-termination-log",
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
			},
			{
				Name:  consts.ControllerServerName,
				Image: args.getContainerImage(),
				Args:  containerArgs,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: commonContainerPorts,
				ReadinessProbe: &corev1.Probe{
					FailureThreshold:    5,
					InitialDelaySeconds: 60,
					TimeoutSeconds:      10,
					PeriodSeconds:       10,
					ProbeHandler:        readinessHandler,
				},
				Env: []corev1.EnvVar{kubeNodeNameEnvVar, csiEndpointEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					k8s.NewVolumeMount(csiDirVolumeName, csiDirVolumePath, corev1.MountPropagationNone, false),
				},
			},
		},
	}

	var selectorValue string
	if !args.DryRun {
		deployment, err := t.client.Kube().AppsV1().Deployments(namespace).Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if err == nil {
			if deployment.Spec.Selector != nil && deployment.Spec.Selector.MatchLabels != nil {
				selectorValue = deployment.Spec.Selector.MatchLabels[selectorKey]
			}
		}
	}
	if selectorValue == "" {
		selectorValue = fmt.Sprintf("%v-%v", consts.ControllerServerName, name)
	}

	replicas := int32(3)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				string(directpvtypes.ImageTagLabelKey): args.imageTag,
			},
			Labels: defaultLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, selectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Annotations: map[string]string{
						createdByLabel: pluginName,
					},
					Labels: map[string]string{
						selectorKey: selectorValue,
					},
				},
				Spec: podSpec,
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
		Status: appsv1.DeploymentStatus{},
	}

	if !args.DryRun && !args.Declarative {
		_, err = t.client.Kube().AppsV1().Deployments(namespace).Create(
			ctx, deployment, metav1.CreateOptions{},
		)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	return args.writeObject(deployment)
}

func (t deploymentTask) createDeployment(ctx context.Context, args *Args) (err error) {
	if err := t.doCreateDeployment(ctx, args, false, 1); err != nil {
		return err
	}

	if args.Legacy {
		if err := t.doCreateDeployment(ctx, args, true, 2); err != nil {
			return err
		}
	}

	return nil
}

func removeFinalizer(objectMeta *metav1.ObjectMeta, finalizer string) []string {
	removeByIndex := func(s []string, index int) []string {
		return append(s[:index], s[index+1:]...)
	}
	finalizers := objectMeta.GetFinalizers()
	for index, f := range finalizers {
		if f == finalizer {
			finalizers = removeByIndex(finalizers, index)
			break
		}
	}
	return finalizers
}

func (t deploymentTask) doDeleteDeployment(ctx context.Context, name string) error {
	deploymentClient := t.client.Kube().AppsV1().Deployments(namespace)

	deployment, err := deploymentClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
		return err
	}

	deployment.SetFinalizers(removeFinalizer(&deployment.ObjectMeta, deploymentFinalizer))
	if _, err = deploymentClient.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return err
	}

	if err = deploymentClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (t deploymentTask) deleteDeployment(ctx context.Context) error {
	if err := t.doDeleteDeployment(ctx, consts.ControllerServerName); err != nil {
		return err
	}

	return t.doDeleteDeployment(ctx, consts.LegacyControllerServerName)
}
