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
	"errors"
	"fmt"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errStorageClassVersionUnsupported = errors.New("unsupported StorageClass version found")

func installStorageClassDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, "storage class", createStorageClassDefault); err != nil {
		return fmt.Errorf("unable to create storage class; %v", err)
	}
	return nil
}

func uninstallStorageClassDefault(ctx context.Context, c *Config) error {
	if err := executeFn(ctx, c, "storage class", deleteStorageClassDefault); err != nil {
		return fmt.Errorf("unable to delete storage class; %v", err)
	}
	return nil
}

func createStorageClassDefault(ctx context.Context, c *Config) error {
	allowExpansion := false
	name := c.storageClassName()
	allowTopologiesWithName := corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			{
				Key:    string(directpvtypes.TopologyDriverIdentity),
				Values: []string{string(directpvtypes.NewLabelValue(c.driverIdentity()))},
			},
		},
	}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
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
				Name:        name,
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

		if !c.DryRun {
			if _, err := k8s.KubeClient().StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
		return c.postProc(storageClass)
	case "v1beta1":
		// Create StorageClass for the new driver
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1beta1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
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

		if !c.DryRun {
			if _, err := k8s.KubeClient().StorageV1beta1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}
		return c.postProc(storageClass)
	default:
		return errStorageClassVersionUnsupported
	}
}

func deleteStorageClassDefault(ctx context.Context, c *Config) error {
	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	name := c.storageClassName()
	switch gvk.Version {
	case "v1":
		if err := k8s.KubeClient().StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	case "v1beta1":
		if err := k8s.KubeClient().StorageV1beta1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	default:
		return errStorageClassVersionUnsupported
	}
	return nil
}
