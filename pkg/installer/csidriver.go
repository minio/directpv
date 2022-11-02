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

	"github.com/minio/directpv/pkg/k8s"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errCSIDriverVersionUnsupported = errors.New("unsupported CSIDriver version found")

func installCSIDriverDefault(ctx context.Context, c *Config) error {
	if err := createCSIDriver(ctx, c); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func uninstallCSIDriverDefault(ctx context.Context, c *Config) error {
	if err := deleteCSIDriver(ctx, c); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func createCSIDriver(ctx context.Context, c *Config) error {
	podInfoOnMount := true
	attachRequired := false

	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}
	version := gvk.Version

	switch version {
	case "v1":
		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        c.csiDriverName(),
				Namespace:   metav1.NamespaceNone,
				Annotations: defaultAnnotations,
				Labels:      defaultLabels,
			},
			Spec: storagev1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
					storagev1.VolumeLifecyclePersistent,
					storagev1.VolumeLifecycleEphemeral,
				},
			},
		}

		if !c.DryRun {
			// Create CSIDriver Obj
			if _, err := k8s.KubeClient().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
				return err
			}
		}
		return c.postProc(csiDriver, "installed '%s' CSI Driver %s", bold(c.csiDriverName()), tick)

	case "v1beta1":
		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1beta1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        c.csiDriverName(),
				Namespace:   metav1.NamespaceNone,
				Annotations: defaultAnnotations,
				Labels:      defaultLabels,
			},
			Spec: storagev1beta1.CSIDriverSpec{
				PodInfoOnMount: &podInfoOnMount,
				AttachRequired: &attachRequired,
				VolumeLifecycleModes: []storagev1beta1.VolumeLifecycleMode{
					storagev1beta1.VolumeLifecyclePersistent,
					storagev1beta1.VolumeLifecycleEphemeral,
				},
			},
		}

		if !c.DryRun {
			// Create CSIDriver Obj
			if _, err := k8s.KubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
				return err
			}
		}
		return c.postProc(csiDriver, "installed '%s' CSI Driver %s", bold(c.csiDriverName()), tick)

	default:
		return errCSIDriverVersionUnsupported
	}
}

func deleteCSIDriver(ctx context.Context, c *Config) error {
	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		// Delete CSIDriver Obj
		if err := k8s.KubeClient().StorageV1().CSIDrivers().Delete(ctx, c.csiDriverName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		// Delete CSIDriver Obj
		if err := k8s.KubeClient().StorageV1beta1().CSIDrivers().Delete(ctx, c.csiDriverName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return errCSIDriverVersionUnsupported
	}
	return c.postProc(nil, "uninstalled '%s' CSI Driver %s", c.csiDriverName(), tick)
}
