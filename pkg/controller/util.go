// This file is part of MinIO Kubernetes Cloud
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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	k8sretry "k8s.io/client-go/util/retry"

	"github.com/golang/glog"
)

func retry(f func() error) error {
	if err := k8sretry.OnError(k8sretry.DefaultBackoff, func(err error) bool {
		if errors.IsAlreadyExists(err) {
			return false
		}
		return true
	}, f); err != nil {
		if !errors.IsAlreadyExists(err) {
			glog.Errorf("CSIDriver creation failed: %v", err)
			return err
		}
	}
	return nil
}

func createCSIDriver(ctx context.Context, kClient clientset.Interface, name string) error {
	podInfoOnMount := false
	attachRequired := false
	csiDriver := &storagev1.CSIDriver{
		metav1.TypeMeta{
			Kind:       "CSIDriver",
			APIVersion: "storage.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
		},
		storagev1.CSIDriverSpec{
			PodInfoOnMount: &podInfoOnMount,
			AttachRequired: &attachRequired,
			VolumeLifecycleModes: []storagev1.VolumeLifecycleMode{
				storagev1.VolumeLifecyclePersistent,
			},
		},
	}

	// Create CSIDriver Obj
	return retry(func() error {
		_, err := kClient.StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		return err
	})
}

func createStorageClass(ctx context.Context, kClient clientset.Interface, name string) error {
	allowExpansion := false
	bindingMode := storagev1.VolumeBindingImmediate
	allowedTopologies := []corev1.TopologySelectorTerm{}
	parameters := map[string]string{}
	mountOptions := []string{}
	retainPolicy := corev1.PersistentVolumeReclaimRetain

	// Create StorageClass for the new driver
	storageClass := &storagev1.StorageClass{
		metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		metav1.ObjectMeta{
			Name: name,
		},
		name, // Provisioner
		parameters,
		&retainPolicy,
		mountOptions,
		&allowExpansion,
		&bindingMode,
		allowedTopologies,
	}

	return retry(func() error {
		_, err := kClient.StorageV1().StorageClasses().Create(ctx, storageClass, metav1.CreateOptions{})
		return err
	})
}
