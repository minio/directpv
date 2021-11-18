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

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var (
	errCSIDriverVersionUnsupported = errors.New("Unsupported CSIDriver version found")
)

func createCSIDriver(ctx context.Context, c *Config) error {
	podInfoOnMount := true
	attachRequired := false

	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
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

		if c.DryRun {
			return c.postProc(csiDriver)
		}

		// Create CSIDriver Obj
		if _, err := client.GetKubeClient().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}

		return c.postProc(csiDriver)

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

		if c.DryRun {
			return c.postProc(csiDriver)
		}

		// Create CSIDriver Obj
		if _, err := client.GetKubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{}); err != nil {
			return err
		}

		return c.postProc(csiDriver)

	default:
		return errCSIDriverVersionUnsupported
	}
}

func deleteCSIDriver(ctx context.Context, c *Config) error {
	gvk, err := client.GetGroupKindVersions("storage.k8s.io", "CSIDriver", "v1", "v1beta1", "v1alpha1")
	if err != nil {
		return err
	}

	switch gvk.Version {
	case "v1":
		// Delete CSIDriver Obj
		if err := client.GetKubeClient().StorageV1().CSIDrivers().Delete(ctx, c.csiDriverName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	case "v1beta1":
		// Delete CSIDriver Obj
		if err := client.GetKubeClient().StorageV1beta1().CSIDrivers().Delete(ctx, c.csiDriverName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	default:
		return errCSIDriverVersionUnsupported
	}
	return nil
}

func installCSIDriverDefault(ctx context.Context, c *Config) error {
	if err := createCSIDriver(ctx, c); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return err
		}
	}

	if !c.DryRun {
		klog.Infof("'%s' csidriver created", utils.Bold(c.Identity))
	}

	return nil
}

func uninstallCSIDriverDefault(ctx context.Context, c *Config) error {
	if err := deleteCSIDriver(ctx, c); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	klog.Infof("'%s' csidriver deleted", utils.Bold(c.Identity))

	return nil
}
