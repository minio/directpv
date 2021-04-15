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

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateStorageClass(ctx context.Context, identity string, dryRun bool) error {
	allowExpansion := false
	allowedTopologies := []corev1.TopologySelectorTerm{}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	version := "v1"
	if !dryRun {
		gvk, err := utils.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	switch version {
	case "v1":
		bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		// Create StorageClass for the new driver
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta:           newObjMeta(identity, ""),
			Provisioner:          sanitizeName(identity),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies:    allowedTopologies,
			ReclaimPolicy:        &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if dryRun {
			return utils.LogYAML(storageClass)
		}

		if _, err := utils.GetKubeClient().
			StorageV1().
			StorageClasses().
			Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		// Create StorageClass for the new driver
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1beta1",
			},
			ObjectMeta:           newObjMeta(identity, ""),
			Provisioner:          sanitizeName(identity),
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies:    allowedTopologies,
			ReclaimPolicy:        &retainPolicy,
			Parameters: map[string]string{
				"fstype": "xfs",
			},
		}

		if dryRun {
			return utils.LogYAML(storageClass)
		}

		if _, err := utils.GetKubeClient().
			StorageV1beta1().
			StorageClasses().
			Create(ctx, storageClass, metav1.CreateOptions{}); err != nil {
			return err
		}
	default:
		return ErrKubeVersionNotSupported
	}
	return nil
}
