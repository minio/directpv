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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func registerDriveValidationRules(ctx context.Context, c *Config) error {
	driveValidatingWebhookConfig := getDriveValidatingWebhookConfig(c)
	if !c.DryRun {
		if _, err := utils.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx, &driveValidatingWebhookConfig, metav1.CreateOptions{}); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
	}
	return c.postProc(driveValidatingWebhookConfig)
}

func getDriveValidatingWebhookConfig(c *Config) admissionv1.ValidatingWebhookConfiguration {

	getServiceRef := func() *admissionv1.ServiceReference {
		path := "/validatedrive"
		return &admissionv1.ServiceReference{
			Namespace: c.namespace(),
			Name:      validationControllerName,
			Path:      &path,
		}
	}

	getClientConfig := func() admissionv1.WebhookClientConfig {
		return admissionv1.WebhookClientConfig{
			Service:  getServiceRef(),
			CABundle: c.validationWebhookCaBundle,
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
		supportedReviewVersions := []string{"v1", "v1beta1", "v1beta2", "v1beta3"}
		sideEffectClass := admissionv1.SideEffectClassNone
		return []admissionv1.ValidatingWebhook{
			{
				Name:                    validationWebhookConfigName,
				ClientConfig:            getClientConfig(),
				AdmissionReviewVersions: supportedReviewVersions,
				SideEffects:             &sideEffectClass,
				Rules:                   getValidationRules(),
			},
		}
	}

	validatingWebhookConfiguration := admissionv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        validationWebhookConfigName,
			Namespace:   c.namespace(),
			Annotations: defaultAnnotations,
			Labels:      defaultLabels,
			Finalizers:  []string{c.namespace() + directCSIFinalizerDeleteProtection},
		},
		Webhooks: getValidatingWebhooks(),
	}

	return validatingWebhookConfiguration
}

func deleteDriveValidationRules(ctx context.Context, c *Config) error {
	vClient := utils.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations()

	getDeleteProtectionFinalizer := func() string {
		return c.namespace() + directCSIFinalizerDeleteProtection
	}

	clearFinalizers := func() error {
		config, err := vClient.Get(ctx, validationWebhookConfigName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		finalizer := getDeleteProtectionFinalizer()
		config.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&config.ObjectMeta, finalizer))
		if _, err := vClient.Update(ctx, config, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}

	if err := clearFinalizers(); err != nil {
		return err
	}

	if err := vClient.Delete(ctx, validationWebhookConfigName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func installValidationRulesDefault(ctx context.Context, c *Config) error {
	if !c.AdmissionControl {
		return nil
	}

	if err := registerDriveValidationRules(ctx, c); err != nil {
		return err
	}

	if !c.DryRun {
		klog.Infof("'%s' validation rules registered", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallValidationRulesDefault(ctx context.Context, c *Config) error {
	if err := deleteDriveValidationRules(ctx, c); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' validation rules removed", utils.Bold(c.Identity))
	return nil
}
