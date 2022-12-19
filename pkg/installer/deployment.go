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
)

const (
	deploymentFinalizer        = consts.Identity + "/delete-protection"
	adminServerCertsDir        = "admin-server-certs"
	adminServerCertsSecretName = "adminservercerts"
	adminServerCASecretName    = "adminservercacert"
	adminServerSelectorValue   = "admin-server"
	nodeAPIServerCADir         = "node-api-server-ca"
)

func doCreateDeployment(ctx context.Context, args *Args, legacy bool) error {
	name := consts.ControllerServerName
	containerArgs := []string{name, fmt.Sprintf("--identity=%s", consts.Identity)}
	if legacy {
		name = consts.LegacyControllerServerName
		containerArgs = []string{name}
	}
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
			newHostPathVolume(
				volumeNameSocketDir,
				newPluginsSocketDir(kubeletDirPath, fmt.Sprintf("%s-controller", consts.ControllerServerName)),
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
				Image: args.getContainerImage(),
				Args:  containerArgs,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports:          commonContainerPorts,
				ReadinessProbe: &corev1.Probe{ProbeHandler: readinessHandler},
				Env:            []corev1.EnvVar{kubeNodeNameEnvVar, csiEndpointEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(volumeNameSocketDir, socketDir, corev1.MountPropagationNone, false),
				},
			},
		},
	}

	selectorValue := fmt.Sprintf("%v-%v", consts.ControllerServerName, getRandSuffix())
	replicas := int32(3)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
			Finalizers:  []string{deploymentFinalizer},
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
		},
		Status: appsv1.DeploymentStatus{},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(deployment))
		return nil
	}

	_, err := k8s.KubeClient().AppsV1().Deployments(namespace).Create(
		ctx, deployment, metav1.CreateOptions{},
	)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(deployment))
	return err
}

func createDeployment(ctx context.Context, args *Args) error {
	if err := doCreateDeployment(ctx, args, false); err != nil {
		return err
	}

	if args.Legacy {
		return doCreateDeployment(ctx, args, true)
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

func doDeleteDeployment(ctx context.Context, name string) error {
	deploymentClient := k8s.KubeClient().AppsV1().Deployments(namespace)

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

	return deploymentClient.Delete(ctx, name, metav1.DeleteOptions{})
}

func deleteDeployment(ctx context.Context) error {
	if err := doDeleteDeployment(ctx, consts.ControllerServerName); err != nil {
		return err
	}

	return doDeleteDeployment(ctx, consts.LegacyControllerServerName)
}

func createAdminService(ctx context.Context, args *Args) error {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "admin-service",
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: consts.AdminServerPort,
					Name: consts.AdminServerPortName,
				},
			},
			Selector: map[string]string{
				serviceSelector: adminServerSelectorValue,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	if args.DryRun {
		fmt.Print(mustGetYAML(service))
		return nil
	}

	_, err := k8s.KubeClient().CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			err = nil
		}
		return err
	}

	_, err = io.WriteString(args.auditWriter, mustGetYAML(service))
	return err
}

func createAdminServerSecrets(ctx context.Context, args *Args) error {
	caCertBytes, publicCertBytes, privateKeyBytes, err := getCerts(
		localhost,
		// FIXME: Add nodeport service domain name here
	)
	if err != nil {
		return err
	}

	err = createOrUpdateSecret(
		ctx,
		args,
		adminServerCertsSecretName,
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
		adminServerCASecretName,
		map[string][]byte{caCertFileName: caCertBytes},
	)
}

func createAdminServerDeployment(ctx context.Context, args *Args) error {
	// Create cert secrets for the admin-server
	if err := createAdminServerSecrets(ctx, args); err != nil {
		return err
	}

	// Create admin-server deployment
	privileged := false
	podSpec := corev1.PodSpec{
		ServiceAccountName: consts.Identity,
		Volumes: []corev1.Volume{
			newSecretVolume(adminServerCertsDir, adminServerCertsSecretName),
			newSecretVolume(nodeAPIServerCADir, nodeAPIServerCASecretName),
		},
		ImagePullSecrets: args.getImagePullSecrets(),
		Containers: []corev1.Container{
			{
				Name:  consts.AdminServerName,
				Image: args.getContainerImage(),
				Args: []string{
					consts.AdminServerName,
					fmt.Sprintf("-v=%d", logLevel),
					fmt.Sprintf("--identity=%s", consts.Identity),
					fmt.Sprintf("--port=%d", consts.AdminServerPort),
					fmt.Sprintf("--csi-endpoint=$(%s)", csiEndpointEnvVarName),
					fmt.Sprintf("--kube-node-name=$(%s)", kubeNodeNameEnvVarName),
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Env: []corev1.EnvVar{kubeNodeNameEnvVar},
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(adminServerCertsDir, consts.AdminServerCertsPath, corev1.MountPropagationNone, false),
					newVolumeMount(nodeAPIServerCADir, "/tmp/nodeapiserver/ca", corev1.MountPropagationNone, false),
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: consts.AdminServerPort,
						Name:          "api-port",
						Protocol:      corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	selectorValue := fmt.Sprintf("%v-%v", consts.AdminServerName, getRandSuffix())
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        consts.AdminServerName,
			Namespace:   namespace,
			Annotations: map[string]string{},
			Labels:      defaultLabels,
			Finalizers:  []string{deploymentFinalizer},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: metav1.AddLabelToSelector(&metav1.LabelSelector{}, selectorKey, selectorValue),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.AdminServerName,
					Namespace: namespace,
					Annotations: map[string]string{
						createdByLabel: pluginName,
					},
					Labels: map[string]string{
						selectorKey:     selectorValue,
						serviceSelector: adminServerSelectorValue,
					},
				},
				Spec: podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}

	if !args.DryRun {
		_, err := k8s.KubeClient().AppsV1().Deployments(namespace).Get(
			ctx, consts.AdminServerName, metav1.GetOptions{},
		)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}

			_, err = k8s.KubeClient().AppsV1().Deployments(namespace).Create(
				ctx, deployment, metav1.CreateOptions{},
			)
			if err != nil {
				return err
			}

			if _, err = io.WriteString(args.auditWriter, mustGetYAML(deployment)); err != nil {
				return err
			}
		}
	} else {
		fmt.Print(mustGetYAML(deployment))
	}

	if !args.DisableAdminService {
		return createAdminService(ctx, args)
	}

	return nil
}

func deleteAdminServerDeployment(ctx context.Context) error {
	if err := doDeleteDeployment(ctx, consts.AdminServerName); err != nil {
		return err
	}
	if err := deleteSecret(ctx, adminServerCertsSecretName); err != nil {
		return err
	}
	if err := deleteSecret(ctx, adminServerCASecretName); err != nil {
		return err
	}

	err := k8s.KubeClient().CoreV1().Services(namespace).Delete(ctx, "admin-service", metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
