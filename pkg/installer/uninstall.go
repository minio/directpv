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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteNamespace(ctx context.Context, identity string) error {
	// Delete Namespace Obj
	if err := utils.GetKubeClient().CoreV1().Namespaces().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteCSIDriver(ctx context.Context, identity string) error {
	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	csiDriver := sanitizeName(identity)
	switch gvk.Version {
	case "v1":
		// Delete CSIDriver Obj
		if err := utils.GetKubeClient().StorageV1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		// Delete CSIDriver Obj
		if err := utils.GetKubeClient().StorageV1beta1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func DeleteStorageClass(ctx context.Context, identity string) error {
	gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		if err := utils.GetKubeClient().StorageV1().StorageClasses().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		if err := utils.GetKubeClient().StorageV1beta1().StorageClasses().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func DeleteService(ctx context.Context, identity string) error {
	if err := utils.GetKubeClient().CoreV1().Services(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteDaemonSet(ctx context.Context, identity string) error {
	if err := utils.GetKubeClient().AppsV1().DaemonSets(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteDriveValidationRules(ctx context.Context, identity string) error {
	vClient := utils.GetKubeClient().AdmissionregistrationV1().ValidatingWebhookConfigurations()

	getDeleteProtectionFinalizer := func() string {
		return sanitizeName(identity) + DirectCSIFinalizerDeleteProtection
	}

	clearFinalizers := func() error {
		config, err := vClient.Get(ctx, ValidationWebhookConfigName, metav1.GetOptions{})
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

	if err := vClient.Delete(ctx, ValidationWebhookConfigName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteControllerSecret(ctx context.Context, identity string) error {
	if err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Delete(ctx, AdmissionWebhookSecretName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteControllerDeployment(ctx context.Context, identity string) error {
	return DeleteDeployment(ctx, identity, sanitizeName(identity))
}

func DeleteConversionDeployment(ctx context.Context, identity string) error {
	return DeleteDeployment(ctx, identity, conversionWebhookName)
}

func DeleteDeployment(ctx context.Context, identity, name string) error {
	dClient := utils.GetKubeClient().AppsV1().Deployments(sanitizeName(identity))

	getDeleteProtectionFinalizer := func() string {
		return sanitizeName(identity) + DirectCSIFinalizerDeleteProtection
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

func DeleteConversionSecret(ctx context.Context, identity string) error {
	if err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Delete(ctx, ConversionWebhookSecretName, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteConversionWebhookCertsSecret(ctx context.Context, identity string) error {
	if err := utils.GetKubeClient().CoreV1().Secrets(sanitizeName(identity)).Delete(ctx, conversionWebhookCertsSecret, metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
