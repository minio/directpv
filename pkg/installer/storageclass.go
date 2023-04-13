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
	"github.com/minio/directpv/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	errStorageClassVersionUnsupported = errors.New("unsupported StorageClass version found")

	existingStorageClassV1      *storagev1.StorageClass
	existingStorageClassV1beta1 *storagev1beta1.StorageClass
)

type storageClassTask struct{}

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

func (storageClassTask) Execute(ctx context.Context, args *Args) error {
	return createStorageClass(ctx, args)
}

func (storageClassTask) Delete(ctx context.Context, _ *Args) error {
	return deleteStorageClass(ctx)
}

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
		update := false
		creationTimestamp := metav1.Time{}
		resourceVersion := ""
		uid := types.UID("")
		if !args.dryRun() {
			existingStorageClassV1, err = k8s.KubeClient().StorageV1().StorageClasses().Get(
				ctx, name, metav1.GetOptions{},
			)
			switch {
			case err != nil:
				if !apierrors.IsNotFound(err) {
					return err
				}
			default:
				if existingStorageClassV1.Provisioner != consts.Identity {
					return fmt.Errorf("legacy storage class with provisioner %v must be uninstalled", existingStorageClassV1.Provisioner)
				}

				update = true
				creationTimestamp = existingStorageClassV1.CreationTimestamp
				resourceVersion = existingStorageClassV1.ResourceVersion
				uid = existingStorageClassV1.UID
				if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingStorageClassV1)); err != nil {
					return err
				}
			}
		}

		bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: creationTimestamp,
				ResourceVersion:   resourceVersion,
				UID:               uid,
				Name:              name,
				Namespace:         metav1.NamespaceNone,
				Annotations:       map[string]string{},
				Labels:            defaultLabels,
				Finalizers:        []string{metav1.FinalizerDeleteDependents},
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

		if args.dryRun() {
			args.DryRunPrinter(storageClass)
			return nil
		}

		if update {
			_, err = k8s.KubeClient().StorageV1().StorageClasses().Update(
				ctx, storageClass, metav1.UpdateOptions{},
			)
		} else {
			_, err = k8s.KubeClient().StorageV1().StorageClasses().Create(
				ctx, storageClass, metav1.CreateOptions{},
			)
		}
		if err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(storageClass))
		return err

	case "v1beta1":
		update := false
		creationTimestamp := metav1.Time{}
		resourceVersion := ""
		uid := types.UID("")
		if !args.dryRun() {
			existingStorageClassV1beta1, err := k8s.KubeClient().StorageV1beta1().StorageClasses().Get(
				ctx, name, metav1.GetOptions{},
			)
			switch {
			case err != nil:
				if !apierrors.IsNotFound(err) {
					return err
				}
			default:
				if existingStorageClassV1beta1.Provisioner != consts.Identity {
					return fmt.Errorf("legacy storage class with provisioner %v must be uninstalled", existingStorageClassV1beta1.Provisioner)
				}

				update = true
				creationTimestamp = existingStorageClassV1beta1.CreationTimestamp
				resourceVersion = existingStorageClassV1beta1.ResourceVersion
				uid = existingStorageClassV1beta1.UID
				if _, err = io.WriteString(args.backupWriter, utils.MustGetYAML(existingStorageClassV1beta1)); err != nil {
					return err
				}
			}
		}

		bindingMode := storagev1beta1.VolumeBindingWaitForFirstConsumer
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{APIVersion: "storage.k8s.io/v1beta1", Kind: "StorageClass"},
			ObjectMeta: metav1.ObjectMeta{
				CreationTimestamp: creationTimestamp,
				ResourceVersion:   resourceVersion,
				UID:               uid,
				Name:              name,
				Namespace:         metav1.NamespaceNone,
				Annotations:       map[string]string{},
				Labels:            defaultLabels,
				Finalizers:        []string{metav1.FinalizerDeleteDependents},
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

		if args.dryRun() {
			args.DryRunPrinter(storageClass)
			return nil
		}

		if update {
			_, err = k8s.KubeClient().StorageV1beta1().StorageClasses().Update(
				ctx, storageClass, metav1.UpdateOptions{},
			)
		} else {
			_, err = k8s.KubeClient().StorageV1beta1().StorageClasses().Create(
				ctx, storageClass, metav1.CreateOptions{},
			)
		}
		if err != nil {
			return err
		}

		_, err = io.WriteString(args.auditWriter, utils.MustGetYAML(storageClass))
		return err

	default:
		return errStorageClassVersionUnsupported
	}
}

func createStorageClass(ctx context.Context, args *Args) (err error) {
	version := "v1"
	switch {
	case args.dryRun():
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

func doRevertStorageClass(ctx context.Context, name string) error {
	switch {
	case existingStorageClassV1 != nil:
		storageClass, err := k8s.KubeClient().StorageV1().StorageClasses().Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil {
			return err
		}

		existingStorageClassV1.ResourceVersion = storageClass.ResourceVersion
		_, err = k8s.KubeClient().StorageV1().StorageClasses().Update(ctx, existingStorageClassV1, metav1.UpdateOptions{})
		return err
	case existingStorageClassV1beta1 != nil:
		storageClass, err := k8s.KubeClient().StorageV1beta1().StorageClasses().Get(
			ctx, name, metav1.GetOptions{},
		)
		if err != nil {
			return err
		}

		existingStorageClassV1beta1.ResourceVersion = storageClass.ResourceVersion
		_, err = k8s.KubeClient().StorageV1beta1().StorageClasses().Update(ctx, existingStorageClassV1beta1, metav1.UpdateOptions{})
		return err
	}

	return nil
}

func revertStorageClass(ctx context.Context, args *Args) error {
	var err, legacyErr error
	err = doRevertStorageClass(ctx, consts.Identity)
	if args.Legacy {
		legacyErr = doRevertStorageClass(ctx, legacyclient.Identity)
	}

	if err != nil && legacyErr != nil {
		return fmt.Errorf("unable to revert; StorageClass: %v; legacy StorageClass: %v", err, legacyErr)
	}

	if err != nil {
		return err
	}

	return legacyErr
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
