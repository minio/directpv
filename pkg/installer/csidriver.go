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

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errCSIDriverVersionUnsupported = errors.New("unsupported CSIDriver version found")

func doCreateCSIDriver(ctx context.Context, args *Args, version string, legacy bool) error {
	podInfoOnMount := true
	attachRequired := false

	name := consts.Identity
	if legacy {
		name = legacyclient.Identity
	}

	switch version {
	case "v1":
		csiDriver := &storagev1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   metav1.NamespaceNone,
				Annotations: map[string]string{},
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

		if args.DryRun {
			fmt.Print(mustGetYAML(csiDriver))
			return nil
		}

		_, err := k8s.KubeClient().StorageV1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				err = nil
			}
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(csiDriver))
		return err

	case "v1beta1":
		csiDriver := &storagev1beta1.CSIDriver{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "storage.k8s.io/v1beta1",
				Kind:       "CSIDriver",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   metav1.NamespaceNone,
				Annotations: map[string]string{},
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

		if args.DryRun {
			fmt.Print(mustGetYAML(csiDriver))
			return nil
		}

		_, err := k8s.KubeClient().StorageV1beta1().CSIDrivers().Create(ctx, csiDriver, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				err = nil
			}
			return err
		}

		_, err = io.WriteString(args.auditWriter, mustGetYAML(csiDriver))
		return err

	default:
		return errCSIDriverVersionUnsupported
	}
}

func createCSIDriver(ctx context.Context, args *Args) error {
	version := "v1"
	if args.DryRun {
		if args.KubeVersion.Major() >= 1 && args.KubeVersion.Minor() < 19 {
			version = "v1beta1"
		}
	} else {
		gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
		if err != nil {
			return err
		}
		version = gvk.Version
	}

	if err := doCreateCSIDriver(ctx, args, version, false); err != nil {
		return err
	}

	if args.Legacy {
		return doCreateCSIDriver(ctx, args, version, true)
	}

	return nil
}

func doDeleteCSIDriver(ctx context.Context, version, name string) (err error) {
	switch version {
	case "v1":
		err = k8s.KubeClient().StorageV1().CSIDrivers().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	case "v1beta1":
		err = k8s.KubeClient().StorageV1beta1().CSIDrivers().Delete(
			ctx, name, metav1.DeleteOptions{},
		)
	default:
		return errCSIDriverVersionUnsupported
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func deleteCSIDriver(ctx context.Context) error {
	gvk, err := k8s.GetGroupVersionKind("storage.k8s.io", "CSIDriver", "v1", "v1beta1")
	if err != nil {
		return err
	}

	if err = doDeleteCSIDriver(ctx, gvk.Version, consts.Identity); err != nil {
		return err
	}

	return doDeleteCSIDriver(ctx, gvk.Version, legacyclient.Identity)
}
