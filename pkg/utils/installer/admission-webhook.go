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

	"github.com/minio/direct-csi/pkg/utils"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func CreateAdmissionWebhookControllerService(ctx context.Context, selector map[string]string, identity string, dryRun bool) error {
	ns := sanitizeName(identity)
	admissionWebhookPort := corev1.ServicePort{
		Port: admissionControllerWebhookPort,
		TargetPort: intstr.IntOrString{
			Type:   intstr.String,
			StrVal: admissionControllerWebhookName,
		},
	}
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: newObjMeta(validationControllerName, identity),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				admissionWebhookPort,
			},
			Selector: selector,
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

func CreateAdmissionWebhookControllerSecret(ctx context.Context, identity string, publicCertBytes, privateKeyBytes []byte, dryRun bool) error {
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
		ObjectMeta: newObjMeta(AdmissionWebhookSecretName, identity),
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

func getDriveValidatingWebhookConfig(identity string, validationWebhookCaBundle []byte) admissionv1.ValidatingWebhookConfiguration {
	name := sanitizeName(identity)
	getServiceRef := func() *admissionv1.ServiceReference {
		path := "/validatedrive"
		return &admissionv1.ServiceReference{
			Namespace: name,
			Name:      validationControllerName,
			Path:      &path,
		}
	}

	getClientConfig := func() admissionv1.WebhookClientConfig {
		return admissionv1.WebhookClientConfig{
			Service:  getServiceRef(),
			CABundle: []byte(validationWebhookCaBundle),
		}

	}

	getValidationRules := func() []admissionv1.RuleWithOperations {
		return []admissionv1.RuleWithOperations{
			{
				Operations: []admissionv1.OperationType{admissionv1.Update},
				Rule: admissionv1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"*"},
					Resources:   []string{"directcsidrives"},
				},
			},
		}
	}

	getValidatingWebhooks := func() []admissionv1.ValidatingWebhook {
		supportedReviewVersions := []string{"v1", "v1beta1"}
		sideEffectClass := admissionv1.SideEffectClassNone
		return []admissionv1.ValidatingWebhook{
			{
				Name:                    ValidationWebhookConfigName,
				ClientConfig:            getClientConfig(),
				AdmissionReviewVersions: supportedReviewVersions,
				SideEffects:             &sideEffectClass,
				Rules:                   getValidationRules(),
			},
		}
	}

	objM := newObjMeta(ValidationWebhookConfigName, name)
	objM.Finalizers = []string{
		sanitizeName(identity) + DirectCSIFinalizerDeleteProtection,
	}
	validatingWebhookConfiguration := admissionv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: objM,
		Webhooks:   getValidatingWebhooks(),
	}

	return validatingWebhookConfiguration
}

func RegisterAdmissionWebhookValidationRules(ctx context.Context, identity string, caCertBytes []byte, dryRun bool) error {
	driveValidatingWebhookConfig := getDriveValidatingWebhookConfig(identity, caCertBytes)
	if dryRun {
		return utils.LogYAML(driveValidatingWebhookConfig)
	}

	if _, err := utils.GetKubeClient().
		AdmissionregistrationV1().
		ValidatingWebhookConfigurations().
		Create(ctx, &driveValidatingWebhookConfig, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
