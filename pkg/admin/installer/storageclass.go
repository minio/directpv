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
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/consts"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errStorageClassVersionUnsupported = errors.New("unsupported StorageClass version found")

type storageClassTask struct {
	client *client.Client
}

func (storageClassTask) Name() string {
	return "StorageClass"
}

func (storageClassTask) Start(ctx context.Context, args *Args) error {
	steps := 1
	if args.Legacy {
		steps++
	}
	if !sendStartMessage(ctx, args.ProgressCh, steps) {
		return errSendProgress
	}
	return nil
}

func (storageClassTask) End(ctx context.Context, args *Args, err error) error {
	if !sendEndMessage(ctx, args.ProgressCh, err) {
		return errSendProgress
	}
	return nil
}

func (t storageClassTask) Execute(ctx context.Context, args *Args) error {
	return t.createStorageClass(ctx, args)
}

func (t storageClassTask) Delete(ctx context.Context, _ *Args) error {
	return t.deleteStorageClass(ctx)
}

func (t storageClassTask) doCreateStorageClass(ctx context.Context, args *Args, version string, legacy bool, step int) (err error) {
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

	allowExpansion := true
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
				Name:      name,
				Namespace: metav1.NamespaceNone,
				Annotations: map[string]string{
					string(directpvtypes.PluginVersionLabelKey): args.PluginVersion,
				},
				Labels:     defaultLabels,
				Finalizers: []string{metav1.FinalizerDeleteDependents},
			},
			Provisioner:          consts.Identity,
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters:    map[string]string{"csi.storage.k8s.io/fstype": "xfs"},
		}

		if !args.DryRun && !args.Declarative {
			_, err := t.client.Kube().StorageV1().StorageClasses().Create(
				ctx, storageClass, metav1.CreateOptions{},
			)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}

		return args.writeObject(storageClass)

	case "v1beta1":
		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1beta1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: metav1.NamespaceNone,
				Annotations: map[string]string{
					string(directpvtypes.PluginVersionLabelKey): args.PluginVersion,
				},
				Labels:     defaultLabels,
				Finalizers: []string{metav1.FinalizerDeleteDependents},
			},
			Provisioner:          consts.Identity,
			AllowVolumeExpansion: &allowExpansion,
			VolumeBindingMode:    &bindingMode,
			AllowedTopologies: []corev1.TopologySelectorTerm{
				allowTopologiesWithName,
			},
			ReclaimPolicy: &retainPolicy,
			Parameters:    map[string]string{"csi.storage.k8s.io/fstype": "xfs"},
		}

		if !args.DryRun && !args.Declarative {
			_, err := t.client.Kube().StorageV1beta1().StorageClasses().Create(
				ctx, storageClass, metav1.CreateOptions{},
			)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return err
			}
		}

		return args.writeObject(storageClass)

	default:
		return errStorageClassVersionUnsupported
	}
}

func (t storageClassTask) createStorageClass(ctx context.Context, args *Args) (err error) {
	version := "v1"
	switch {
	case args.DryRun:
		if args.KubeVersion.Major() >= 1 && args.KubeVersion.Minor() < 16 {
			version = "v1beta1"
		}
	default:
		gvk, err := t.client.K8s().GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	if err := t.doCreateStorageClass(ctx, args, version, false, 1); err != nil {
		return err
	}

	if args.Legacy {
		if err := t.doCreateStorageClass(ctx, args, version, true, 2); err != nil {
			return err
		}
	}

	return nil
}

func (t storageClassTask) doDeleteStorageClass(ctx context.Context, version, name string) (err error) {
	switch version {
	case "v1":
		err = t.client.Kube().StorageV1().StorageClasses().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	case "v1beta1":
		err = t.client.Kube().StorageV1beta1().StorageClasses().Delete(
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

func (t storageClassTask) deleteStorageClass(ctx context.Context) error {
	gvk, err := t.client.K8s().GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
	if err != nil {
		return err
	}

	if err = t.doDeleteStorageClass(ctx, gvk.Version, consts.StorageClassName); err != nil {
		return err
	}

	return t.doDeleteStorageClass(ctx, gvk.Version, legacyclient.Identity)
}
