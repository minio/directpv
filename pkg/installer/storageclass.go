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
	"io"

	directpvtypes "github.com/minio/directpv/pkg/apis/directpv.min.io/types"
	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	totalStorageClassSteps = 2
)

var errStorageClassVersionUnsupported = errors.New("unsupported StorageClass version found")

func doCreateStorageClass(ctx context.Context, args *Args, version string, legacy bool, step int) (err error) {
	name := consts.Identity
	if legacy {
		name = legacyclient.Identity
	}
	if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Creating %s Storage Class", name), step, nil) {
		return errSendProgress
	}
	defer func() {
		if err == nil {
			if !sendProgressMessage(ctx, args.ProgressCh, fmt.Sprintf("Created %s Storage Class", name), step, storageClassComponent(name)) {
				err = errSendProgress
			}
		}
	}()

	allowExpansion := false
	allowTopologiesWithName := corev1.TopologySelectorTerm{
		MatchLabelExpressions: []corev1.TopologySelectorLabelRequirement{
			{
				Key:    string(directpvtypes.TopologyDriverIdentity),
				Values: []string{consts.Identity},
			},
		},
	}
	retainPolicy := corev1.PersistentVolumeReclaimDelete

	switch version {
	case "v1":
		bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   metav1.NamespaceNone,
				Annotations: map[string]string{},
				Labels:      defaultLabels,
				Finalizers:  []string{metav1.FinalizerDeleteDependents},
			},
			Provisioner:          consts.Identity,
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters:    map[string]string{"fstype": "xfs"},
		}

		if args.DryRun {
			fmt.Print(mustGetYAML(storageClass))
			return nil
		}

		_, err := k8s.KubeClient().StorageV1().StorageClasses().Create(
			ctx, storageClass, metav1.CreateOptions{},
		)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				err = nil
			}
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(storageClass))
		return err

	case "v1beta1":
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1beta1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   metav1.NamespaceNone,
				Annotations: map[string]string{},
				Labels:      defaultLabels,
				Finalizers:  []string{metav1.FinalizerDeleteDependents},
			},
			Provisioner:          consts.Identity,
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters:    map[string]string{"fstype": "xfs"},
		}

		if args.DryRun {
			fmt.Print(mustGetYAML(storageClass))
			return nil
		}

		_, err := k8s.KubeClient().StorageV1beta1().StorageClasses().Create(
			ctx, storageClass, metav1.CreateOptions{},
		)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				err = nil
			}
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(storageClass))
		return err

	default:
		return errStorageClassVersionUnsupported
	}
}

func createStorageClass(ctx context.Context, args *Args) (err error) {
	version := "v1"
	switch {
	case args.DryRun:
		if args.KubeVersion.Major() >= 1 && args.KubeVersion.Minor() < 16 {
			version = "v1beta1"
		}
	default:
		gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	if err := doCreateStorageClass(ctx, args, version, false, 1); err != nil {
		return err
	}

	if args.Legacy {
		if err := doCreateStorageClass(ctx, args, version, true, 2); err != nil {
			return err
		}
	}

	return nil
}

func doDeleteStorageClass(ctx context.Context, version, name string) (err error) {
	switch version {
	case "v1":
		err = k8s.KubeClient().StorageV1().StorageClasses().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	case "v1beta1":
		err = k8s.KubeClient().StorageV1beta1().StorageClasses().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	default:
		return errStorageClassVersionUnsupported
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func deleteStorageClass(ctx context.Context) error {
	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
	if err != nil {
		return err
	}

	if err = doDeleteStorageClass(ctx, gvk.Version, consts.StorageClassName); err != nil {
		return err
	}

	return doDeleteStorageClass(ctx, gvk.Version, legacyclient.Identity)
}
