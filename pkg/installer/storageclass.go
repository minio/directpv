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
	"errors"

	"github.com/minio/direct-csi/pkg/client"
	"github.com/minio/direct-csi/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var (
	errStorageClassVersionUnsupported = errors.New("Unsupported StorageClass version found")
)

func installStorageClassDefault(ctx context.Context, c *Config) error {
	if err := createStorageClass(ctx, c); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	if !c.DryRun {
		klog.Infof("'%s' storageclass created", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallStorageClassDefault(ctx context.Context, c *Config) error {
	if err := deleteStorageClass(ctx, c); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' storageclass deleted", utils.Bold(c.Identity))
	return nil
}

func createStorageClass(ctx context.Context, c *Config) error {
	allowExpansion := false
	allowTopologiesWithName := corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			{
				Key:    string(utils.TopologyDriverIdentity),
				Values: []string{string(utils.NewLabelValue(c.driverIdentity()))},
			},
		},
	}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	version := gvk.Version

	switch version {
	case "v1":
		// Create StorageClass for the new driver
		bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        c.storageClassName(),
				Namespace:   metav1.NamespaceNone,
				Annotations: defaultAnnotations,
				Labels:      defaultLabels,
				Finalizers:  []string{metav1.FinalizerDeleteDependents}, // foregroundDeletion finalizer
			},
			Provisioner:          c.provisionerName(),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if c.DryRun {
			return c.postProc(storageClass)
		}

		if _, err := client.GetKubeClient().StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
		return c.postProc(storageClass)
	case "v1beta1":
		// Create StorageClass for the new driver
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1beta1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        c.storageClassName(),
				Namespace:   metav1.NamespaceNone,
				Annotations: defaultAnnotations,
				Labels:      defaultLabels,
				Finalizers:  []string{metav1.FinalizerDeleteDependents}, // foregroundDeletion finalizer
			},
			Provisioner:          c.provisionerName(),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if c.DryRun {
			return c.postProc(storageClass)
		}

		if _, err := client.GetKubeClient().StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
		return c.postProc(storageClass)
	default:
		return errStorageClassVersionUnsupported
	}
}

func deleteStorageClass(ctx context.Context, c *Config) error {
	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		if err := client.GetKubeClient().StorageV1().StorageClasses().Delete(ctx, c.storageClassName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		if err := client.GetKubeClient().StorageV1beta1().StorageClasses().Delete(ctx, c.storageClassName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return errStorageClassVersionUnsupported
	}
	return nil
}
