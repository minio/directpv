// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteNamespace(ctx context.Context, identity string) error {
	// Delete Namespace Obj
	if err := kubeClient.CoreV1().Namespaces().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteCSIDriver(ctx context.Context, identity string) error {
	gvk, err := GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	csiDriver := sanitizeName(identity)
	switch gvk.Version {
	case "v1":
		// Delete CSIDriver Obj
		if err := kubeClient.StorageV1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		// Delete CSIDriver Obj
		if err := kubeClient.StorageV1beta1().CSIDrivers().Delete(ctx, csiDriver, metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func DeleteStorageClass(ctx context.Context, identity string) error {
	gvk, err := GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		if err := kubeClient.StorageV1().StorageClasses().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		if err := kubeClient.StorageV1beta1().StorageClasses().Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}

func DeleteService(ctx context.Context, identity string) error {
	if err := kubeClient.CoreV1().Services(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteDaemonSet(ctx context.Context, identity string) error {
	if err := kubeClient.AppsV1().DaemonSets(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteDeployment(ctx context.Context, identity string) error {
	if err := kubeClient.AppsV1().Deployments(sanitizeName(identity)).Delete(ctx, sanitizeName(identity), metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
