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

	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteNamespace deletes direct-csi namespace.
func DeleteNamespace(ctx context.Context, identity string) error {
	// Delete Namespace Obj
	if err := client.GetKubeClient().CoreV1().Namespaces().Delete(ctx, utils.SanitizeKubeResourceName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// DeleteCSIDriver deletes direct-csi driver.
func DeleteCSIDriver(ctx context.Context, identity string) error {
	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	csiDriver := utils.SanitizeKubeResourceName(identity)
	switch gvk.Version {
	case "v1":
		// Delete CSIDriver Obj
		if err := client.GetKubeClient().StorageV1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		// Delete CSIDriver Obj
		if err := client.GetKubeClient().StorageV1beta1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

// DeleteStorageClass deletes storage class.
func DeleteStorageClass(ctx context.Context, identity string) error {
	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		if err := client.GetKubeClient().StorageV1().StorageClasses().Delete(ctx, utils.SanitizeKubeResourceName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		if err := client.GetKubeClient().StorageV1beta1().StorageClasses().Delete(ctx, utils.SanitizeKubeResourceName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

// DeleteService deletes service.
func DeleteService(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().CoreV1().Services(utils.SanitizeKubeResourceName(identity)).Delete(ctx, utils.SanitizeKubeResourceName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// DeleteDaemonSet deletes direct-csi daemonset.
func DeleteDaemonSet(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().AppsV1().DaemonSets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, utils.SanitizeKubeResourceName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// DeleteDriveValidationRules deletes drive validation rules.
func DeleteDriveValidationRules(ctx context.Context, identity string) error {
	vClient := client.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations()

	getDeleteProtectionFinalizer := func() string {
		return utils.SanitizeKubeResourceName(identity) + directCSIFinalizerDeleteProtection
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

// DeleteControllerSecret deletes controller secret.
func DeleteControllerSecret(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, admissionWebhookSecretName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// DeleteControllerDeployment deletes controller deployment.
func DeleteControllerDeployment(ctx context.Context, identity string) error {
	return DeleteDeployment(ctx, identity, utils.SanitizeKubeResourceName(identity))
}

func deleteConversionDeployment(ctx context.Context, identity string) error {
	return DeleteDeployment(ctx, identity, conversionWebhookDeploymentName)
}

// DeleteDeployment deletes deployment.
func DeleteDeployment(ctx context.Context, identity, name string) error {
	dClient := client.GetKubeClient().AppsV1().Deployments(utils.SanitizeKubeResourceName(identity))

	getDeleteProtectionFinalizer := func() string {
		return utils.SanitizeKubeResourceName(identity) + directCSIFinalizerDeleteProtection
	}

	clearFinalizers := func(name string) error {
		deployment, err := dClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		finalizer := getDeleteProtectionFinalizer()
		deployment.ObjectMeta.SetFinalizers(utils.RemoveFinalizer(&deployment.ObjectMeta, finalizer))
		if _, err := dClient.Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}

	if err := clearFinalizers(name); err != nil {
		return err
	}

	if err := dClient.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func deleteLegacyConversionSecret(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, conversionWebhookSecretName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func deleteLegacyConversionWebhookCertsSecret(ctx context.Context, identity string) error {
	if err := client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity)).Delete(ctx, conversionWebhookCertsSecret, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// DeleteConversionSecrets deletes conversion secrets.
func DeleteConversionSecrets(ctx context.Context, identity string) error {
	secretsClient := client.GetKubeClient().CoreV1().Secrets(utils.SanitizeKubeResourceName(identity))
	if err := secretsClient.Delete(ctx, conversionKeyPair, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return secretsClient.Delete(ctx, conversionCACert, metav1.DeleteOptions{})
}

// DeleteLegacyConversionDeployment deletes legacy conversion deployment.
func DeleteLegacyConversionDeployment(ctx context.Context, identity string) error {
	if err := deleteConversionDeployment(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := deleteLegacyConversionSecret(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := deleteLegacyConversionWebhookCertsSecret(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// DeletePodSecurityPolicy deletes pod security policy.
func DeletePodSecurityPolicy(ctx context.Context, identity string) error {
	if err := removePSPClusterRoleBinding(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := deletePodSecurityPolicy(ctx, identity); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
