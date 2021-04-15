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
	"net/url"
	"path/filepath"
	"strings"

	"github.com/minio/direct-csi/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateConversionWebhookCASecret(ctx context.Context, identity string, caCertBytes []byte, dryRun bool) error {
	ns := sanitizeName(identity)
	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[caCertFileName] = caCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: newObjMeta(conversionWebhookCertsSecret, identity),
		Data:       getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().
		CoreV1().
		Secrets(ns).
		Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionWebhookSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte, dryRun bool) error {
	ns := sanitizeName(identity)
	getCertsDataMap := func() map[string][]byte {
		mp := make(map[string][]byte)
		mp[privateKeyFileName] = privateKeyBytes
		mp[publicCertFileName] = publicCertBytes
		return mp
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConversionWebhookSecretName,
			Namespace: sanitizeName(identity),
		},
		Data: getCertsDataMap(),
	}

	if dryRun {
		return utils.LogYAML(secret)
	}

	if _, err := utils.GetKubeClient().
		CoreV1().
		Secrets(ns).
		Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionWebhookService(ctx context.Context, labels map[string]string, identity string, dryRun bool) error {
	ns := sanitizeName(identity)
	webhookPort := corev1.ServicePort{
		Port: conversionWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: conversionWebhookPortName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: newObjMeta(conversionWebhookName, identity, "component", "conversion-webhook"),
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{webhookPort},
			Selector: labels,
		},
	}

	if dryRun {
		return utils.LogYAML(svc)
	}

	if _, err := utils.GetKubeClient().
		CoreV1().
		Services(ns).
		Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func CreateConversionWebhookDeployment(
	ctx context.Context,
	identity string,
	directCSIContainerImage string,
	dryRun bool,
	registry, org string) error {

	name := sanitizeName(identity)
	ns := sanitizeName(identity)

	var replicas int32 = 3
	privileged := true
	podSpec := corev1.PodSpec{
		ServiceAccountName: name,
		Volumes: []corev1.Volume{
			newSecretVolume(conversionWebhookName, ConversionWebhookSecretName),
		},
		Containers: []corev1.Container{
			{
				Name:  directCSIContainerName,
				Image: filepath.Join(registry, org, directCSIContainerImage),
				Args: []string{
					"--conversion-webhook",
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: conversionWebhookPort,
						Name:          conversionWebhookPortName,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				// // Enable after investigating the CrashLoopBackOff on pods
				// LivenessProbe: &corev1.Probe{
				// 	FailureThreshold:    5,
				// 	InitialDelaySeconds: 10,
				// 	TimeoutSeconds:      3,
				// 	PeriodSeconds:       2,
				// 	Handler: corev1.Handler{
				// 		HTTPGet: &corev1.HTTPGetAction{
				// 			Path:   healthZContainerPortPath,
				// 			Port:   intstr.FromString(conversionWebhookPortName),
				// 			Host:   conversionWebhookDNSName,
				// 			Scheme: corev1.URISchemeHTTPS,
				// 		},
				// 	},
				// },
				VolumeMounts: []corev1.VolumeMount{
					newVolumeMount(conversionWebhookName, certsDir, false),
				},
			},
		},
	}

	objMeta := newObjMeta(conversionWebhookName, name, "component", "conversion-webhook")
	selector := &metav1.LabelSelector{}
	for k, v := range objMeta.Labels {
		selector = metav1.AddLabelToSelector(selector, k, v)
	}
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: objMeta,
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: objMeta,
				Spec:       podSpec,
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
	deployment.ObjectMeta.Finalizers = []string{
		sanitizeName(identity) + DirectCSIFinalizerDeleteProtection,
	}

	if dryRun {
		return utils.LogYAML(deployment)
	}

	if _, err := utils.GetKubeClient().
		AppsV1().
		Deployments(ns).
		Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
func GetConversionWebhookServiceName() string {
	return conversionWebhookName
}

func getConversionWebhookDNSName(identity string) string {
	components := []string{
		conversionWebhookName,  // directcsi-conversion-webhook
		sanitizeName(identity), // direct-csi-min-io
		"svc",                  // svc
	}
	return strings.Join(components, ".") // "directcsi-conversion-webhook.direct-csi-min-io.svc"
}

func getConversionWebhookURL(identity string) string {
	conversionWebhookDNSName := getConversionWebhookDNSName(identity)
	conversionWebhookURL := url.URL{
		Host:   conversionWebhookDNSName,
		Path:   healthZContainerPortPath,
		Scheme: "https",
	}
	return conversionWebhookURL.String()
}
